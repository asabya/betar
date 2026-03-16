package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asabya/betar/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for Betar",
	Long:  "Walk through LLM provider, wallet, and agent setup. Writes config.yaml (network) and agents.yaml (agent profiles).",
	RunE:  runOnboard,
}

func runOnboard(cmd *cobra.Command, args []string) error {
	dataDir := os.Getenv("BETAR_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".betar")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("cannot create data directory: %w", err)
	}

	cfgPath := config.FileConfigPath(dataDir)

	// Load existing configs for pre-populating defaults on re-run
	existingFC, _ := config.LoadFileConfig(cfgPath)
	if existingFC == nil {
		existingFC = &config.FileConfig{}
	}
	existingAgents, _ := config.LoadAgentsConfig(dataDir)
	if existingAgents == nil {
		existingAgents = &config.AgentsConfig{}
	}

	// Collect LLM provider info (will be stored in agent profile, not config.yaml)
	var provider, apiKey, baseURL, model string

	// --- Step 1: LLM Provider ---
	// Pre-populate from first existing agent profile
	var existingProfile *config.AgentProfile
	if len(existingAgents.Agents) > 0 {
		existingProfile = &existingAgents.Agents[0]
	}

	provider = "google"
	if existingProfile != nil && existingProfile.Provider != "" {
		provider = existingProfile.Provider
	}
	if err := huh.NewSelect[string]().
		Title("Select your AI provider").
		Options(
			huh.NewOption("Google (Gemini)", "google"),
			huh.NewOption("OpenAI / Ollama", "openai"),
		).
		Value(&provider).
		Run(); err != nil {
		return err
	}

	apiKeyDefault := ""
	if existingProfile != nil {
		if provider == "openai" {
			apiKeyDefault = existingProfile.OpenAIAPIKey
		} else {
			apiKeyDefault = existingProfile.APIKey
		}
	}
	if err := huh.NewInput().
		Title("Enter your API key").
		Value(&apiKey).
		Placeholder(maskKey(apiKeyDefault)).
		Run(); err != nil {
		return err
	}
	if apiKey == "" {
		apiKey = apiKeyDefault
	}

	if provider == "openai" {
		baseURLDefault := "https://api.openai.com/v1/"
		if existingProfile != nil && existingProfile.OpenAIBaseURL != "" {
			baseURLDefault = existingProfile.OpenAIBaseURL
		}
		if err := huh.NewInput().
			Title("Base URL").
			Description("For Ollama use http://localhost:11434/v1/").
			Value(&baseURL).
			Placeholder(baseURLDefault).
			Run(); err != nil {
			return err
		}
		if baseURL == "" {
			baseURL = baseURLDefault
		}
	}

	modelDefault := ""
	if existingProfile != nil && existingProfile.Model != "" {
		modelDefault = existingProfile.Model
	}
	if modelDefault == "" {
		if provider == "google" {
			modelDefault = "gemini-2.5-flash"
		} else {
			modelDefault = "gpt-4o"
		}
	}
	if err := huh.NewInput().
		Title("Model").
		Value(&model).
		Placeholder(modelDefault).
		Run(); err != nil {
		return err
	}
	if model == "" {
		model = modelDefault
	}

	// --- Step 2: Wallet ---
	walletKeyPath := filepath.Join(dataDir, "wallet.key")
	walletExists := fileExists(walletKeyPath)

	walletChoice := "skip"
	walletOptions := []huh.Option[string]{
		huh.NewOption("Generate new wallet", "generate"),
		huh.NewOption("Import existing private key", "import"),
		huh.NewOption("Skip for now", "skip"),
	}
	if walletExists {
		walletOptions = []huh.Option[string]{
			huh.NewOption("Keep existing wallet", "keep"),
			huh.NewOption("Generate new wallet (replaces current)", "generate"),
			huh.NewOption("Import existing private key", "import"),
			huh.NewOption("Skip for now", "skip"),
		}
		walletChoice = "keep"
	}

	if err := huh.NewSelect[string]().
		Title("Set up an Ethereum wallet for payments?").
		Options(walletOptions...).
		Value(&walletChoice).
		Run(); err != nil {
		return err
	}

	switch walletChoice {
	case "generate":
		if walletExists {
			var confirm bool
			if err := huh.NewConfirm().
				Title("This will replace your existing wallet key. Any funds in the old wallet will be inaccessible. Continue?").
				Affirmative("Yes, replace").
				Negative("No, keep existing").
				Value(&confirm).
				Run(); err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Keeping existing wallet.")
				break
			}
		}
		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate wallet key: %w", err)
		}
		keyHex := fmt.Sprintf("%x", crypto.FromECDSA(key))
		if err := os.WriteFile(walletKeyPath, []byte(keyHex+"\n"), 0o600); err != nil {
			return fmt.Errorf("failed to save wallet key: %w", err)
		}
		addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
		fmt.Printf("Generated wallet: %s\n", addr)
		fmt.Printf("Private key saved to %s\n", walletKeyPath)

	case "import":
		var keyHex string
		if err := huh.NewInput().
			Title("Enter your private key (hex, without 0x prefix)").
			Value(&keyHex).
			EchoMode(huh.EchoModePassword).
			Run(); err != nil {
			return err
		}
		keyHex = strings.TrimPrefix(strings.TrimSpace(keyHex), "0x")
		if _, err := crypto.HexToECDSA(keyHex); err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}
		if err := os.WriteFile(walletKeyPath, []byte(keyHex+"\n"), 0o600); err != nil {
			return fmt.Errorf("failed to save wallet key: %w", err)
		}
		fmt.Println("Wallet key imported.")

	case "keep":
		fmt.Println("Keeping existing wallet.")
	}

	rpcDefault := existingFC.RPCUrl
	if rpcDefault == "" {
		rpcDefault = "https://sepolia.base.org"
	}
	var rpcURL string
	if err := huh.NewInput().
		Title("Ethereum RPC URL").
		Value(&rpcURL).
		Placeholder(rpcDefault).
		Run(); err != nil {
		return err
	}
	if rpcURL == "" {
		rpcURL = rpcDefault
	}

	// --- Step 3: Agent Profile ---
	var setupAgent bool
	if err := huh.NewConfirm().
		Title("Set up a default agent?").
		Affirmative("Yes").
		Negative("No").
		Value(&setupAgent).
		Run(); err != nil {
		return err
	}

	if setupAgent {
		nameDefault := "my-agent"
		if existingProfile != nil && existingProfile.Name != "" {
			nameDefault = existingProfile.Name
		}
		var agentName string
		if err := huh.NewInput().
			Title("Agent name").
			Value(&agentName).
			Placeholder(nameDefault).
			Run(); err != nil {
			return err
		}
		if agentName == "" {
			agentName = nameDefault
		}

		descDefault := ""
		if existingProfile != nil {
			descDefault = existingProfile.Description
		}
		var agentDesc string
		if err := huh.NewInput().
			Title("Agent description").
			Value(&agentDesc).
			Placeholder(descDefault).
			Run(); err != nil {
			return err
		}
		if agentDesc == "" {
			agentDesc = descDefault
		}

		priceDefault := "0"
		if existingProfile != nil && existingProfile.Price > 0 {
			priceDefault = fmt.Sprintf("%g", existingProfile.Price)
		}
		var priceStr string
		if err := huh.NewInput().
			Title("Price per task in USDC").
			Value(&priceStr).
			Placeholder(priceDefault).
			Run(); err != nil {
			return err
		}
		if priceStr == "" {
			priceStr = priceDefault
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return fmt.Errorf("invalid price %q: %w", priceStr, err)
		}

		// Build agent profile with provider info
		profile := config.AgentProfile{
			Name:        agentName,
			Description: agentDesc,
			Price:       price,
			Provider:    provider,
			Model:       model,
		}
		if provider == "openai" {
			profile.OpenAIAPIKey = apiKey
			profile.OpenAIBaseURL = baseURL
		} else {
			profile.APIKey = apiKey
		}

		// Upsert into agents.yaml — update existing or add new
		if existing := existingAgents.FindProfile(agentName); existing != nil {
			*existing = profile
		} else {
			if err := existingAgents.AddProfile(profile); err != nil {
				return fmt.Errorf("failed to add agent profile: %w", err)
			}
		}

		if err := config.SaveAgentsConfig(dataDir, existingAgents); err != nil {
			return fmt.Errorf("failed to save agents config: %w", err)
		}
		fmt.Printf("Agent profile saved to %s\n", config.AgentsConfigPath(dataDir))
	}

	// --- Save flat config.yaml (network settings only) ---
	fc := &config.FileConfig{
		RPCUrl:         rpcURL,
		P2PPort:        4001,
		BootstrapPeers: existingFC.BootstrapPeers,
	}
	if existingFC.P2PPort != 0 {
		fc.P2PPort = existingFC.P2PPort
	}

	if err := config.SaveFileConfig(cfgPath, fc); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("Config saved to %s\n", cfgPath)
	fmt.Println()
	fmt.Println("Get started:")
	fmt.Println("  betar start          # Start your agent")
	fmt.Println("  betar onboard        # Re-run setup anytime")

	return nil
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
