// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title ReputationRegistry
 * @dev EIP-8004 Feedback Registry
 *      Tracks agent reputation, ratings, and task history
 */
contract ReputationRegistry is Ownable, ReentrancyGuard {
    // Reputation data for each agent
    struct ReputationData {
        uint256 totalTasks;
        uint256 successfulTasks;
        uint256 totalRating;
        uint256 ratingCount;
        uint256 totalEarnings;
        uint256 lastUpdated;
    }

    // Feedback entry
    struct Feedback {
        address from;
        uint256 rating; // 1-5
        string comment; // IPFS CID
        uint256 timestamp;
    }

    // Agent address to ReputationData mapping
    mapping(uint256 => ReputationData) public reputationData;

    // Agent to feedback list
    mapping(uint256 => Feedback[]) public feedbackList;

    // Mapping to track if address has rated an agent
    mapping(uint256 => mapping(address => bool)) public hasRated;

    // Events
    event TaskCompleted(uint256 indexed agentId, bool success, uint256 earnings);
    event FeedbackSubmitted(uint256 indexed agentId, address from, uint256 rating);
    event ReputationUpdated(uint256 indexed agentId, uint256 newRating);

    constructor() Ownable(msg.sender) {}

    /**
     * @dev Record task completion
     */
    function recordTaskCompletion(
        uint256 agentId,
        bool success,
        uint256 earnings
    ) external nonReentrant {
        ReputationData storage rep = reputationData[agentId];

        rep.totalTasks++;
        if (success) {
            rep.successfulTasks++;
        }
        rep.totalEarnings += earnings;
        rep.lastUpdated = block.timestamp;

        emit TaskCompleted(agentId, success, earnings);
    }

    /**
     * @dev Submit feedback for an agent
     */
    function submitFeedback(
        uint256 agentId,
        uint256 rating,
        string memory comment
    ) external {
        require(rating >= 1 && rating <= 5, "Rating must be between 1 and 5");
        require(!hasRated[agentId][msg.sender], "Already rated this agent");

        ReputationData storage rep = reputationData[agentId];

        // Record feedback
        feedbackList[agentId].push(Feedback({
            from: msg.sender,
            rating: rating,
            comment: comment,
            timestamp: block.timestamp
        }));

        // Update rating
        rep.totalRating += rating;
        rep.ratingCount++;
        rep.lastUpdated = block.timestamp;

        // Mark as rated
        hasRated[agentId][msg.sender] = true;

        emit FeedbackSubmitted(agentId, msg.sender, rating);
        emit ReputationUpdated(agentId, getAverageRating(agentId));
    }

    /**
     * @dev Get reputation data for an agent
     */
    function getReputation(uint256 agentId) external view returns (
        uint256 totalTasks,
        uint256 successfulTasks,
        uint256 averageRating,
        uint256 ratingCount,
        uint256 totalEarnings
    ) {
        ReputationData storage rep = reputationData[agentId];
        return (
            rep.totalTasks,
            rep.successfulTasks,
            getAverageRating(agentId),
            rep.ratingCount,
            rep.totalEarnings
        );
    }

    /**
     * @dev Get success rate for an agent
     */
    function getSuccessRate(uint256 agentId) external view returns (uint256) {
        ReputationData storage rep = reputationData[agentId];
        if (rep.totalTasks == 0) {
            return 0;
        }
        return (rep.successfulTasks * 100) / rep.totalTasks;
    }

    /**
     * @dev Get average rating for an agent
     */
    function getAverageRating(uint256 agentId) public view returns (uint256) {
        ReputationData storage rep = reputationData[agentId];
        if (rep.ratingCount == 0) {
            return 0;
        }
        return rep.totalRating / rep.ratingCount;
    }

    /**
     * @dev Get feedback count for an agent
     */
    function getFeedbackCount(uint256 agentId) external view returns (uint256) {
        return feedbackList[agentId].length;
    }

    /**
     * @dev Get feedback for an agent by index
     */
    function getFeedback(uint256 agentId, uint256 index) external view returns (
        address from,
        uint256 rating,
        string memory comment,
        uint256 timestamp
    ) {
        require(index < feedbackList[agentId].length, "Invalid index");
        Feedback storage fb = feedbackList[agentId][index];
        return (fb.from, fb.rating, fb.comment, fb.timestamp);
    }

    /**
     * @dev Check if an address has rated an agent
     */
    function hasUserRated(uint256 agentId, address user) external view returns (bool) {
        return hasRated[agentId][user];
    }
}
