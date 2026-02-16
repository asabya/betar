// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title ValidationRegistry
 * @dev EIP-8004 Validation Registry
 *      Handles agent validation, verification, and compliance
 */
contract ValidationRegistry is Ownable, ReentrancyGuard {
    // Validation status enum
    enum ValidationStatus {
        Unvalidated,
        Pending,
        Validated,
        Rejected,
        Expired
    }

    // Validation record
    struct ValidationRecord {
        address validator;
        uint256 agentId;
        ValidationStatus status;
        uint256 expiresAt;
        string metadata; // IPFS CID with validation details
        uint256 createdAt;
        uint256 updatedAt;
    }

    // Agent ID to validation record
    mapping(uint256 => ValidationRecord) public validationRecords;

    // Validator role management
    mapping(address => bool) public validators;
    mapping(address => uint256) public validatorStake;

    // Minimum stake required to be a validator
    uint256 public constant MIN_VALIDATOR_STAKE = 1 ether;

    // Validation expiry period
    uint256 public constant VALIDATION_PERIOD = 365 days;

    // Events
    event ValidatorRegistered(address indexed validator, uint256 stake);
    event ValidatorRemoved(address indexed validator);
    event ValidationRequested(uint256 indexed agentId, address requester);
    event ValidationCompleted(uint256 indexed agentId, ValidationStatus status, address validator);
    event ValidationRevoked(uint256 indexed agentId);

    modifier onlyValidator() {
        require(validators[msg.sender], "Not a validator");
        _;
    }

    constructor() Ownable(msg.sender) {}

    /**
     * @dev Register as a validator (requires stake)
     */
    function registerValidator() external payable nonReentrant {
        require(!validators[msg.sender], "Already a validator");
        require(msg.value >= MIN_VALIDATOR_STAKE, "Insufficient stake");

        validators[msg.sender] = true;
        validatorStake[msg.sender] = msg.value;

        emit ValidatorRegistered(msg.sender, msg.value);
    }

    /**
     * @dev Remove validator (returns stake)
     */
    function removeValidator(address validator) external onlyOwner {
        require(validators[validator], "Not a validator");

        uint256 stake = validatorStake[validator];
        validators[validator] = 0;
        validatorStake[validator] = 0;

        // Return stake (use call to prevent reentrancy)
        (bool success, ) = validator.call{value: stake}("");
        require(success, "Stake return failed");

        emit ValidatorRemoved(validator);
    }

    /**
     * @dev Request validation for an agent
     */
    function requestValidation(uint256 agentId, string memory metadata) external nonReentrant {
        require(validationRecords[agentId].status == ValidationStatus.Unvalidated ||
                validationRecords[agentId].status == ValidationStatus.Expired ||
                validationRecords[agentId].status == ValidationStatus.Rejected,
            "Agent already validated or pending");

        validationRecords[agentId] = ValidationRecord({
            validator: address(0),
            agentId: agentId,
            status: ValidationStatus.Pending,
            expiresAt: 0,
            metadata: metadata,
            createdAt: block.timestamp,
            updatedAt: block.timestamp
        });

        emit ValidationRequested(agentId, msg.sender);
    }

    /**
     * @dev Validate an agent
     */
    function validateAgent(
        uint256 agentId,
        ValidationStatus status,
        string memory metadata
    ) external onlyValidator nonReentrant {
        require(validationRecords[agentId].status == ValidationStatus.Pending, "Not pending validation");
        require(status == ValidationStatus.Validated || status == ValidationStatus.Rejected, "Invalid status");

        validationRecords[agentId].validator = msg.sender;
        validationRecords[agentId].status = status;
        validationRecords[agentId].expiresAt = block.timestamp + VALIDATION_PERIOD;
        validationRecords[agentId].metadata = metadata;
        validationRecords[agentId].updatedAt = block.timestamp;

        emit ValidationCompleted(agentId, status, msg.sender);
    }

    /**
     * @dev Revoke validation
     */
    function revokeValidation(uint256 agentId) external onlyValidator nonReentrant {
        require(validationRecords[agentId].status == ValidationStatus.Validated, "Not validated");
        require(validationRecords[agentId].validator == msg.sender || msg.sender == owner(), "Not authorized");

        validationRecords[agentId].status = ValidationStatus.Expired;
        validationRecords[agentId].updatedAt = block.timestamp;

        emit ValidationRevoked(agentId);
    }

    /**
     * @dev Get validation status
     */
    function getValidationStatus(uint256 agentId) external view returns (
        ValidationStatus status,
        address validator,
        uint256 expiresAt,
        bool isValid
    ) {
        ValidationRecord storage rec = validationRecords[agentId];

        // Check if expired
        if (rec.status == ValidationStatus.Validated && rec.expiresAt < block.timestamp) {
            return (ValidationStatus.Expired, rec.validator, rec.expiresAt, false);
        }

        return (rec.status, rec.validator, rec.expiresAt, rec.status == ValidationStatus.Validated);
    }

    /**
     * @dev Check if agent is validated
     */
    function isValidated(uint256 agentId) external view returns (bool) {
        ValidationRecord storage rec = validationRecords[agentId];
        return rec.status == ValidationStatus.Validated && (rec.expiresAt == 0 || rec.expiresAt > block.timestamp);
    }

    /**
     * @dev Get validation record
     */
    function getValidationRecord(uint256 agentId) external view returns (ValidationRecord memory) {
        return validationRecords[agentId];
    }
}
