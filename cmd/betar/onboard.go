package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asabya/betar/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for Betar",
	Long:  "Walk through LLM provider, wallet, and agent setup. Writes ~/.betar/config.yaml.",
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

	// Load existing config for pre-populating defaults on re-run
	existing, _ := config.LoadFileConfig(cfgPath)
	if existing == nil {
		existing = &config.FileConfig{}
	}

	fc := &config.FileConfig{}

	// --- Step 1: LLM Provider ---
	provider := existing.LLM.Provider
	if provider == "" {
		provider = "google"
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
	fc.LLM.Provider = provider

	apiKeyDefault := existing.LLM.APIKey
	var apiKey string
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
	fc.LLM.APIKey = apiKey

	if provider == "openai" {
		baseURLDefault := existing.LLM.BaseURL
		if baseURLDefault == "" {
			baseURLDefault = "https://api.openai.com/v1/"
		}
		var baseURL string
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
		fc.LLM.BaseURL = baseURL
	}

	modelDefault := existing.LLM.Model
	if modelDefault == "" {
		if provider == "google" {
			modelDefault = "gemini-2.5-flash"
		} else {
			modelDefault = "gpt-4o"
		}
	}
	var model string
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
	fc.LLM.Model = model

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

	rpcDefault := existing.Wallet.RPCURL
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
	fc.Wallet.RPCURL = rpcURL

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
		nameDefault := existing.Agent.Name
		if nameDefault == "" {
			nameDefault = "my-agent"
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
		fc.Agent.Name = agentName

		var agentDesc string
		if err := huh.NewInput().
			Title("Agent description").
			Value(&agentDesc).
			Placeholder(existing.Agent.Description).
			Run(); err != nil {
			return err
		}
		if agentDesc == "" {
			agentDesc = existing.Agent.Description
		}
		fc.Agent.Description = agentDesc

		// Price — use string input, parse to float
		priceDefault := "0"
		if existing.Agent.Price > 0 {
			priceDefault = fmt.Sprintf("%g", existing.Agent.Price)
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
		var price float64
		fmt.Sscanf(priceStr, "%f", &price)
		fc.Agent.Price = price
	}

	// P2P defaults
	fc.P2P.Port = 4001
	if existing.P2P.Port != 0 {
		fc.P2P.Port = existing.P2P.Port
	}
	fc.P2P.BootstrapPeers = existing.P2P.BootstrapPeers

	// --- Save ---
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
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
