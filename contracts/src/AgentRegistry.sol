// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721Burnable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title AgentRegistry
 * @dev EIP-8004 Identity Registry for AI Agents
 *      ERC-721 based registry for agent identity and metadata
 */
contract AgentRegistry is ERC721, ERC721URIStorage, ERC721Burnable, Ownable, ReentrancyGuard {
    // Token ID counter
    uint256 private _tokenIds;

    // Agent data struct
    struct Agent {
        string name;
        string description;
        string metadataURI; // IPFS CID
        string[] services;
        bool x402Support;
        bool active;
        uint256 createdAt;
        uint256 updatedAt;
    }

    // Token ID to Agent mapping
    mapping(uint256 => Agent) public agents;

    // Owner to token IDs mapping
    mapping(address => uint256[]) public ownerTokens;

    // Events
    event AgentRegistered(uint256 indexed tokenId, address indexed owner, string name, string metadataURI);
    event AgentUpdated(uint256 indexed tokenId, string metadataURI, bool active);
    event ServiceAdded(uint256 indexed tokenId, string service);
    event ServiceRemoved(uint256 indexed tokenId, string service);

    constructor() ERC721("Betar Agent", "BETA") Ownable(msg.sender) {}

    /**
     * @dev Register a new agent
     */
    function registerAgent(
        string memory name,
        string memory description,
        string memory metadataURI,
        string[] memory services,
        bool x402Support
    ) public nonReentrant returns (uint256) {
        _tokenIds++;
        uint256 newTokenId = _tokenIds;

        _safeMint(msg.sender, newTokenId);

        // Set initial metadata
        _setTokenURI(newTokenId, metadataURI);

        // Store agent data
        agents[newTokenId] = Agent({
            name: name,
            description: description,
            metadataURI: metadataURI,
            services: services,
            x402Support: x402Support,
            active: true,
            createdAt: block.timestamp,
            updatedAt: block.timestamp
        });

        // Track ownership
        ownerTokens[msg.sender].push(newTokenId);

        emit AgentRegistered(newTokenId, msg.sender, name, metadataURI);

        return newTokenId;
    }

    /**
     * @dev Update agent metadata
     */
    function updateAgent(
        uint256 tokenId,
        string memory metadataURI,
        bool active
    ) public {
        require(ownerOf(tokenId) == msg.sender, "Not the agent owner");

        agents[tokenId].metadataURI = metadataURI;
        agents[tokenId].active = active;
        agents[tokenId].updatedAt = block.timestamp;

        _setTokenURI(tokenId, metadataURI);

        emit AgentUpdated(tokenId, metadataURI, active);
    }

    /**
     * @dev Add a service to an agent
     */
    function addService(uint256 tokenId, string memory service) public {
        require(ownerOf(tokenId) == msg.sender, "Not the agent owner");
        agents[tokenId].services.push(service);
        agents[tokenId].updatedAt = block.timestamp;
        emit ServiceAdded(tokenId, service);
    }

    /**
     * @dev Remove a service from an agent
     */
    function removeService(uint256 tokenId, uint256 index) public {
        require(ownerOf(tokenId) == msg.sender, "Not the agent owner");
        require(index < agents[tokenId].services.length, "Invalid index");

        agents[tokenId].services[index] = agents[tokenId].services[agents[tokenId].services.length - 1];
        agents[tokenId].services.pop();
        agents[tokenId].updatedAt = block.timestamp;
        emit ServiceRemoved(tokenId, agents[tokenId].services[index]);
    }

    /**
     * @dev Get agent details
     */
    function getAgent(uint256 tokenId) public view returns (Agent memory) {
        require(_exists(tokenId), "Agent does not exist");
        return agents[tokenId];
    }

    /**
     * @dev Get all token IDs for an owner
     */
    function getOwnerTokens(address owner) public view returns (uint256[] memory) {
        return ownerTokens[owner];
    }

    /**
     * @dev Check if agent supports X402 payments
     */
    function supportsX402(uint256 tokenId) public view returns (bool) {
        require(_exists(tokenId), "Agent does not exist");
        return agents[tokenId].x402Support;
    }

    /**
     * @dev Check if agent is active
     */
    function isActive(uint256 tokenId) public view returns (bool) {
        require(_exists(tokenId), "Agent does not exist");
        return agents[tokenId].active;
    }

    // Required overrides for multiple inheritance
    function tokenURI(uint256 tokenId)
        public view override(ERC721, ERC721URIStorage)
        returns (string memory)
    {
        return super.tokenURI(tokenId);
    }

    function supportsInterface(bytes4 interfaceId)
        public view override(ERC721, ERC721URIStorage)
        returns (bool)
    {
        return super.supportsInterface(interfaceId);
    }
}
