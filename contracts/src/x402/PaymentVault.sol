// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title PaymentVault
 * @dev EIP-402 Payment Vault
 *      Handles payments for agent tasks with escrow functionality
 */
contract PaymentVault is Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    // Payment status
    enum PaymentStatus {
        Pending,
        Released,
        Refunded,
        Cancelled
    }

    // Payment record
    struct Payment {
        bytes32 paymentId;
        address payer;
        address payee;
        address token; // address(0) for ETH
        uint256 amount;
        PaymentStatus status;
        bytes32 orderId;
        uint256 createdAt;
        uint256 releasedAt;
    }

    // Payment ID to Payment mapping
    mapping(bytes32 => Payment) public payments;

    // Order ID to payment ID mapping
    mapping(bytes32 => bytes32[]) public orderPayments;

    // Supported tokens
    mapping(address => bool) public supportedTokens;

    // Platform fee (in basis points, 100 = 1%)
    uint256 public platformFee = 250; // 2.5%

    // Events
    event PaymentCreated(bytes32 indexed paymentId, address payer, address payee, uint256 amount);
    event PaymentReleased(bytes32 indexed paymentId, address recipient, uint256 amount);
    event PaymentRefunded(bytes32 indexed paymentId, address recipient, uint256 amount);
    event PaymentCancelled(bytes32 indexed paymentId);
    event TokenSupported(address indexed token, bool supported);
    event PlatformFeeUpdated(uint256 newFee);

    constructor() Ownable(msg.sender) {
        // Support ETH by default
        supportedTokens[address(0)] = true;
    }

    /**
     * @dev Create a payment
     */
    function createPayment(
        bytes32 paymentId,
        address payee,
        address token,
        uint256 amount,
        bytes32 orderId
    ) external payable nonReentrant {
        require(payments[paymentId].payer == address(0), "Payment already exists");
        require(payee != address(0), "Invalid payee");
        require(supportedTokens[token], "Token not supported");
        require(amount > 0, "Amount must be greater than 0");

        // Handle ETH payment
        if (token == address(0)) {
            require(msg.value >= amount, "Insufficient ETH sent");
        } else {
            // Handle ERC20 payment
            IERC20(token).safeTransferFrom(msg.sender, address(this), amount);
        }

        // Create payment record
        payments[paymentId] = Payment({
            paymentId: paymentId,
            payer: msg.sender,
            payee: payee,
            token: token,
            amount: amount,
            status: PaymentStatus.Pending,
            orderId: orderId,
            createdAt: block.timestamp,
            releasedAt: 0
        });

        // Link to order
        orderPayments[orderId].push(paymentId);

        emit PaymentCreated(paymentId, msg.sender, payee, amount);
    }

    /**
     * @dev Release payment to payee
     */
    function releasePayment(bytes32 paymentId) external nonReentrant {
        Payment storage payment = payments[paymentId];
        require(payment.payer == msg.sender || msg.sender == owner(), "Not authorized");
        require(payment.status == PaymentStatus.Pending, "Payment not pending");

        payment.status = PaymentStatus.Released;
        payment.releasedAt = block.timestamp;

        // Calculate platform fee
        uint256 fee = (payment.amount * platformFee) / 10000;
        uint256 payout = payment.amount - fee;

        // Transfer to payee
        if (payment.token == address(0)) {
            (bool success, ) = payment.payee.call{value: payout}("");
            require(success, "Transfer to payee failed");
        } else {
            IERC20(payment.token).safeTransfer(payment.payee, payout);
        }

        // Transfer fee to owner
        if (fee > 0 && payment.token == address(0)) {
            (bool success, ) = owner().call{value: fee}("");
            require(success, "Fee transfer failed");
        } else if (fee > 0) {
            IERC20(payment.token).safeTransfer(owner(), fee);
        }

        emit PaymentReleased(paymentId, payment.payee, payout);
    }

    /**
     * @dev Refund payment to payer
     */
    function refundPayment(bytes32 paymentId) external nonReentrant {
        Payment storage payment = payments[paymentId];
        require(payment.payee == msg.sender || msg.sender == owner(), "Not authorized");
        require(payment.status == PaymentStatus.Pending, "Payment not pending");

        payment.status = PaymentStatus.Refunded;

        // Refund to payer
        if (payment.token == address(0)) {
            (bool success, ) = payment.payer.call{value: payment.amount}("");
            require(success, "Refund failed");
        } else {
            IERC20(payment.token).safeTransfer(payment.payer, payment.amount);
        }

        emit PaymentRefunded(paymentId, payment.payer, payment.amount);
    }

    /**
     * @dev Cancel payment (only owner)
     */
    function cancelPayment(bytes32 paymentId) external onlyOwner {
        Payment storage payment = payments[paymentId];
        require(payment.status == PaymentStatus.Pending, "Payment not pending");

        payment.status = PaymentStatus.Cancelled;

        // Return funds to payer
        if (payment.token == address(0)) {
            (bool success, ) = payment.payer.call{value: payment.amount}("");
            require(success, "Refund failed");
        } else {
            IERC20(payment.token).safeTransfer(payment.payer, payment.amount);
        }

        emit PaymentCancelled(paymentId);
    }

    /**
     * @dev Add or remove supported token
     */
    function setSupportedToken(address token, bool supported) external onlyOwner {
        supportedTokens[token] = supported;
        emit TokenSupported(token, supported);
    }

    /**
     * @dev Set platform fee
     */
    function setPlatformFee(uint256 newFee) external onlyOwner {
        require(newFee <= 10000, "Fee too high"); // Max 100%
        platformFee = newFee;
        emit PlatformFeeUpdated(newFee);
    }

    /**
     * @dev Get payment details
     */
    function getPayment(bytes32 paymentId) external view returns (Payment memory) {
        return payments[paymentId];
    }

    /**
     * @dev Get payments for an order
     */
    function getOrderPayments(bytes32 orderId) external view returns (bytes32[] memory) {
        return orderPayments[orderId];
    }

    /**
     * @dev Get payment count for an order
     */
    function getOrderPaymentCount(bytes32 orderId) external view returns (uint256) {
        return orderPayments[orderId].length;
    }

    // Receive ETH
    receive() external payable {}
}
