// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import {IdentityRegistryUpgradeable} from "contracts/IdentityRegistryUpgradeable.sol";
import {ReputationRegistryUpgradeable} from "contracts/ReputationRegistryUpgradeable.sol";
import {ValidationRegistryUpgradeable} from "contracts/ValidationRegistryUpgradeable.sol";
import {MockUSDC} from "contracts/MockUSDC.sol";

contract Deploy is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        
        vm.startBroadcast(deployerPrivateKey);
        
        console.log("Deploying IdentityRegistryUpgradeable...");
        IdentityRegistryUpgradeable identityRegistry = new IdentityRegistryUpgradeable();
        console.log("IdentityRegistryUpgradeable deployed at:", address(identityRegistry));
        
        identityRegistry.initialize();
        console.log("IdentityRegistry initialized");
        
        console.log("Deploying ReputationRegistryUpgradeable...");
        ReputationRegistryUpgradeable reputationRegistry = new ReputationRegistryUpgradeable();
        console.log("ReputationRegistryUpgradeable deployed at:", address(reputationRegistry));
        
        reputationRegistry.initialize(address(identityRegistry));
        console.log("ReputationRegistry initialized");
        
        console.log("Deploying ValidationRegistryUpgradeable...");
        ValidationRegistryUpgradeable validationRegistry = new ValidationRegistryUpgradeable();
        console.log("ValidationRegistryUpgradeable deployed at:", address(validationRegistry));
        
        validationRegistry.initialize(address(identityRegistry));
        console.log("ValidationRegistry initialized");
        
        console.log("Deploying MockUSDC...");
        MockUSDC mockUSDC = new MockUSDC();
        console.log("MockUSDC deployed at:", address(mockUSDC));
        
        vm.stopBroadcast();
        
        console.log("\n=== ERC-8004 Deployment Complete ===");
        console.log("Network Chain ID:", block.chainid);
        console.log("Deployer:", vm.addr(deployerPrivateKey));
        console.log("\nContract Addresses:");
        console.log("  IdentityRegistry:   ", address(identityRegistry));
        console.log("  ReputationRegistry: ", address(reputationRegistry));
        console.log("  ValidationRegistry: ", address(validationRegistry));
        console.log("  MockUSDC:          ", address(mockUSDC));
    }
}
