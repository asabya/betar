// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {AgentRegistry} from "../src/AgentRegistry.sol";
import {ReputationRegistry} from "../src/ReputationRegistry.sol";
import {ValidationRegistry} from "../src/ValidationRegistry.sol";
import {PaymentVault} from "../src/x402/PaymentVault.sol";

contract Deploy is Script {
    function run() public {
        uint256 deployerPrivateKey = vm.envUint("ETHEREUM_PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        AgentRegistry agentRegistry = new AgentRegistry();
        console.log("AgentRegistry deployed at:", address(agentRegistry));

        ReputationRegistry reputationRegistry = new ReputationRegistry();
        console.log("ReputationRegistry deployed at:", address(reputationRegistry));

        ValidationRegistry validationRegistry = new ValidationRegistry();
        console.log("ValidationRegistry deployed at:", address(validationRegistry));

        PaymentVault paymentVault = new PaymentVault();
        console.log("PaymentVault deployed at:", address(paymentVault));

        vm.stopBroadcast();

        console.log("\n--- Deployment Summary ---");
        console.log("AGENT_REGISTRY_ADDRESS=%s", address(agentRegistry));
        console.log("REPUTATION_REGISTRY_ADDRESS=%s", address(reputationRegistry));
        console.log("VALIDATION_REGISTRY_ADDRESS=%s", address(validationRegistry));
        console.log("PAYMENT_VAULT_ADDRESS=%s", address(paymentVault));
    }
}
