# Payment Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Payment Service with REST API, gRPC integration, state machine, and full test coverage.

**Architecture:** Layered architecture (Controller → Service → Repository) with DTOs for API contracts, domain entities for persistence, and Kafka for async events. Integrates with Exchange Rate Service via gRPC for real rates.

**Tech Stack:** Java 21, Spring Boot 3.2, PostgreSQL, Kafka, gRPC, JUnit 5, Testcontainers

---

## Phase A: API Layer (DTOs, Exception Handling, Controllers)

### Task 1: Create Transfer DTOs

**Files:**
- Create: `src/main/java/com/movra/payment/dto/CreateTransferRequest.java`
- Create: `src/main/java/com/movra/payment/dto/TransferResponse.java`
- Create: `src/main/java/com/movra/payment/dto/QuoteRequest.java`
- Create: `src/main/java/com/movra/payment/dto/QuoteResponse.java`

**Step 1: Create CreateTransferRequest DTO**

```java
package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import jakarta.validation.constraints.*;
import lombok.Data;

import java.math.BigDecimal;

@Data
public class CreateTransferRequest {

    @NotBlank(message = "Idempotency key is required")
    private String idempotencyKey;

    @NotBlank(message = "Source currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
    private String sourceCurrency;

    @NotNull(message = "Source amount is required")
    @DecimalMin(value = "1.00", message = "Minimum amount is 1.00")
    @DecimalMax(value = "50000.00", message = "Maximum amount is 50,000.00")
    private BigDecimal sourceAmount;

    @NotBlank(message = "Target currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
    private String targetCurrency;

    @NotNull(message = "Funding method is required")
    private FundingMethod fundingMethod;

    @NotNull(message = "Recipient ID is required")
    private String recipientId;

    private String rateLockId;
}
```

**Step 2: Create TransferResponse DTO**

```java
package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import com.movra.payment.model.TransferStatus;
import lombok.Builder;
import lombok.Data;

import java.math.BigDecimal;
import java.time.Instant;

@Data
@Builder
public class TransferResponse {
    private String id;
    private String userId;
    private TransferStatus status;

    private String sourceCurrency;
    private BigDecimal sourceAmount;
    private String targetCurrency;
    private BigDecimal targetAmount;

    private BigDecimal exchangeRate;
    private String rateLockId;
    private Instant rateExpiresAt;

    private BigDecimal feeAmount;
    private String feeCurrency;

    private FundingMethod fundingMethod;
    private PayoutMethod payoutMethod;

    private String recipientId;
    private RecipientResponse recipient;

    private FundingDetailsResponse fundingDetails;

    private Instant createdAt;
    private Instant updatedAt;
    private Instant completedAt;

    private String failureReason;
}
```

**Step 3: Create FundingDetailsResponse DTO**

```java
package com.movra.payment.dto;

import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class FundingDetailsResponse {
    private String bankName;
    private String accountNumber;
    private String accountName;
    private String reference;
}
```

**Step 4: Create QuoteRequest and QuoteResponse DTOs**

```java
package com.movra.payment.dto;

import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import jakarta.validation.constraints.*;
import lombok.Data;

import java.math.BigDecimal;

@Data
public class QuoteRequest {
    @NotBlank(message = "Source currency is required")
    @Size(min = 3, max = 3)
    private String sourceCurrency;

    @NotNull(message = "Source amount is required")
    @DecimalMin(value = "1.00")
    private BigDecimal sourceAmount;

    @NotBlank(message = "Target currency is required")
    @Size(min = 3, max = 3)
    private String targetCurrency;

    @NotNull(message = "Funding method is required")
    private FundingMethod fundingMethod;

    private PayoutMethod payoutMethod;
}
```

```java
package com.movra.payment.dto;

import lombok.Builder;
import lombok.Data;

import java.math.BigDecimal;
import java.time.Instant;

@Data
@Builder
public class QuoteResponse {
    private String sourceCurrency;
    private BigDecimal sourceAmount;
    private String targetCurrency;
    private BigDecimal targetAmount;
    private BigDecimal exchangeRate;
    private BigDecimal fee;
    private BigDecimal totalCost;
    private Instant validUntil;
}
```

**Step 5: Commit**

```bash
git add src/main/java/com/movra/payment/dto/
git commit -m "feat(payment): add transfer and quote DTOs with validation"
```

---

### Task 2: Create Recipient DTOs

**Files:**
- Create: `src/main/java/com/movra/payment/dto/CreateRecipientRequest.java`
- Create: `src/main/java/com/movra/payment/dto/RecipientResponse.java`

**Step 1: Create CreateRecipientRequest DTO**

```java
package com.movra.payment.dto;

import com.movra.payment.model.PayoutMethod;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;
import lombok.Data;

@Data
public class CreateRecipientRequest {

    @NotBlank(message = "Nickname is required")
    @Size(max = 50, message = "Nickname must be 50 characters or less")
    private String nickname;

    @NotNull(message = "Payout method is required")
    private PayoutMethod type;

    @NotBlank(message = "Country is required")
    @Size(min = 2, max = 2, message = "Country must be 2 characters (ISO)")
    private String country;

    @NotBlank(message = "Currency is required")
    @Size(min = 3, max = 3, message = "Currency must be 3 characters")
    private String currency;

    // Bank account details
    private String bankName;
    private String bankCode;
    private String accountNumber;
    private String accountName;

    // Mobile wallet details
    private String walletProvider;
    private String mobileNumber;

    // Cash pickup details
    private String firstName;
    private String lastName;
}
```

**Step 2: Create RecipientResponse DTO**

```java
package com.movra.payment.dto;

import com.movra.payment.model.PayoutMethod;
import lombok.Builder;
import lombok.Data;

import java.time.Instant;

@Data
@Builder
public class RecipientResponse {
    private String id;
    private String userId;
    private String nickname;
    private PayoutMethod type;
    private String country;
    private String currency;

    // Bank account details
    private String bankName;
    private String bankCode;
    private String accountNumber;
    private String accountName;

    // Mobile wallet details
    private String walletProvider;
    private String mobileNumber;

    // Cash pickup details
    private String firstName;
    private String lastName;

    private Instant createdAt;
}
```

**Step 3: Commit**

```bash
git add src/main/java/com/movra/payment/dto/
git commit -m "feat(payment): add recipient DTOs"
```

---

### Task 3: Create API Error Response and Exception Handler

**Files:**
- Create: `src/main/java/com/movra/payment/dto/ApiError.java`
- Create: `src/main/java/com/movra/payment/exception/ResourceNotFoundException.java`
- Create: `src/main/java/com/movra/payment/exception/InvalidStateTransitionException.java`
- Create: `src/main/java/com/movra/payment/exception/RateLockExpiredException.java`
- Create: `src/main/java/com/movra/payment/exception/GlobalExceptionHandler.java`

**Step 1: Create ApiError DTO**

```java
package com.movra.payment.dto;

import lombok.Builder;
import lombok.Data;

import java.time.Instant;
import java.util.Map;

@Data
@Builder
public class ApiError {
    private String code;
    private String message;
    private Map<String, String> details;
    private Instant timestamp;
    private String path;
}
```

**Step 2: Create custom exceptions**

```java
package com.movra.payment.exception;

public class ResourceNotFoundException extends RuntimeException {
    private final String resourceType;
    private final String resourceId;

    public ResourceNotFoundException(String resourceType, String resourceId) {
        super(String.format("%s not found: %s", resourceType, resourceId));
        this.resourceType = resourceType;
        this.resourceId = resourceId;
    }

    public String getResourceType() { return resourceType; }
    public String getResourceId() { return resourceId; }
}
```

```java
package com.movra.payment.exception;

import com.movra.payment.model.TransferStatus;

public class InvalidStateTransitionException extends RuntimeException {
    private final TransferStatus currentStatus;
    private final TransferStatus targetStatus;

    public InvalidStateTransitionException(TransferStatus current, TransferStatus target) {
        super(String.format("Invalid state transition: %s -> %s", current, target));
        this.currentStatus = current;
        this.targetStatus = target;
    }

    public TransferStatus getCurrentStatus() { return currentStatus; }
    public TransferStatus getTargetStatus() { return targetStatus; }
}
```

```java
package com.movra.payment.exception;

public class RateLockExpiredException extends RuntimeException {
    private final String transferId;

    public RateLockExpiredException(String transferId) {
        super("Rate lock has expired for transfer: " + transferId);
        this.transferId = transferId;
    }

    public String getTransferId() { return transferId; }
}
```

**Step 3: Create GlobalExceptionHandler**

```java
package com.movra.payment.exception;

import com.movra.payment.dto.ApiError;
import jakarta.servlet.http.HttpServletRequest;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;

@RestControllerAdvice
@Slf4j
public class GlobalExceptionHandler {

    @ExceptionHandler(ResourceNotFoundException.class)
    public ResponseEntity<ApiError> handleResourceNotFound(
            ResourceNotFoundException ex, HttpServletRequest request) {
        log.warn("Resource not found: {}", ex.getMessage());

        ApiError error = ApiError.builder()
                .code("RESOURCE_NOT_FOUND")
                .message(ex.getMessage())
                .details(Map.of(
                        "resourceType", ex.getResourceType(),
                        "resourceId", ex.getResourceId()
                ))
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(error);
    }

    @ExceptionHandler(InvalidStateTransitionException.class)
    public ResponseEntity<ApiError> handleInvalidStateTransition(
            InvalidStateTransitionException ex, HttpServletRequest request) {
        log.warn("Invalid state transition: {}", ex.getMessage());

        ApiError error = ApiError.builder()
                .code("INVALID_STATE_TRANSITION")
                .message(ex.getMessage())
                .details(Map.of(
                        "currentStatus", ex.getCurrentStatus().toString(),
                        "targetStatus", ex.getTargetStatus().toString()
                ))
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.CONFLICT).body(error);
    }

    @ExceptionHandler(RateLockExpiredException.class)
    public ResponseEntity<ApiError> handleRateLockExpired(
            RateLockExpiredException ex, HttpServletRequest request) {
        log.warn("Rate lock expired: {}", ex.getMessage());

        ApiError error = ApiError.builder()
                .code("RATE_LOCK_EXPIRED")
                .message("Rate lock has expired. Please create a new transfer.")
                .details(Map.of("transferId", ex.getTransferId()))
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.GONE).body(error);
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ApiError> handleValidation(
            MethodArgumentNotValidException ex, HttpServletRequest request) {
        Map<String, String> errors = new HashMap<>();
        ex.getBindingResult().getAllErrors().forEach(error -> {
            String fieldName = ((FieldError) error).getField();
            String errorMessage = error.getDefaultMessage();
            errors.put(fieldName, errorMessage);
        });

        ApiError error = ApiError.builder()
                .code("VALIDATION_FAILED")
                .message("Request validation failed")
                .details(errors)
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(error);
    }

    @ExceptionHandler(IllegalArgumentException.class)
    public ResponseEntity<ApiError> handleIllegalArgument(
            IllegalArgumentException ex, HttpServletRequest request) {
        log.warn("Bad request: {}", ex.getMessage());

        ApiError error = ApiError.builder()
                .code("BAD_REQUEST")
                .message(ex.getMessage())
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(error);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ApiError> handleGeneral(
            Exception ex, HttpServletRequest request) {
        log.error("Unexpected error", ex);

        ApiError error = ApiError.builder()
                .code("INTERNAL_ERROR")
                .message("An unexpected error occurred")
                .timestamp(Instant.now())
                .path(request.getRequestURI())
                .build();

        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(error);
    }
}
```

**Step 4: Commit**

```bash
git add src/main/java/com/movra/payment/dto/ApiError.java
git add src/main/java/com/movra/payment/exception/
git commit -m "feat(payment): add exception handling with global handler"
```

---

### Task 4: Create DTO Mapper

**Files:**
- Create: `src/main/java/com/movra/payment/mapper/TransferMapper.java`
- Create: `src/main/java/com/movra/payment/mapper/RecipientMapper.java`

**Step 1: Create TransferMapper**

```java
package com.movra.payment.mapper;

import com.movra.payment.dto.FundingDetailsResponse;
import com.movra.payment.dto.TransferResponse;
import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.Transfer;
import org.springframework.stereotype.Component;

@Component
public class TransferMapper {

    private final RecipientMapper recipientMapper;

    public TransferMapper(RecipientMapper recipientMapper) {
        this.recipientMapper = recipientMapper;
    }

    public TransferResponse toResponse(Transfer transfer) {
        return TransferResponse.builder()
                .id(transfer.getId().toString())
                .userId(transfer.getUserId().toString())
                .status(transfer.getStatus())
                .sourceCurrency(transfer.getSourceCurrency())
                .sourceAmount(transfer.getSourceAmount())
                .targetCurrency(transfer.getTargetCurrency())
                .targetAmount(transfer.getTargetAmount())
                .exchangeRate(transfer.getExchangeRate())
                .rateLockId(transfer.getRateLockId())
                .rateExpiresAt(transfer.getRateExpiresAt())
                .feeAmount(transfer.getFeeAmount())
                .feeCurrency(transfer.getFeeCurrency())
                .fundingMethod(transfer.getFundingMethod())
                .payoutMethod(transfer.getPayoutMethod())
                .recipientId(transfer.getRecipientId().toString())
                .fundingDetails(buildFundingDetails(transfer))
                .createdAt(transfer.getCreatedAt())
                .updatedAt(transfer.getUpdatedAt())
                .completedAt(transfer.getCompletedAt())
                .failureReason(transfer.getFailureReason())
                .build();
    }

    private FundingDetailsResponse buildFundingDetails(Transfer transfer) {
        if (transfer.getFundingMethod() != FundingMethod.BANK_TRANSFER) {
            return null;
        }
        return FundingDetailsResponse.builder()
                .bankName(transfer.getFundingBankName())
                .accountNumber(transfer.getFundingAccountNumber())
                .accountName(transfer.getFundingAccountName())
                .reference(transfer.getFundingReference())
                .build();
    }
}
```

**Step 2: Create RecipientMapper**

```java
package com.movra.payment.mapper;

import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.dto.RecipientResponse;
import com.movra.payment.model.Recipient;
import org.springframework.stereotype.Component;

@Component
public class RecipientMapper {

    public RecipientResponse toResponse(Recipient recipient) {
        return RecipientResponse.builder()
                .id(recipient.getId().toString())
                .userId(recipient.getUserId().toString())
                .nickname(recipient.getNickname())
                .type(recipient.getType())
                .country(recipient.getCountry())
                .currency(recipient.getCurrency())
                .bankName(recipient.getBankName())
                .bankCode(recipient.getBankCode())
                .accountNumber(recipient.getAccountNumber())
                .accountName(recipient.getAccountName())
                .walletProvider(recipient.getWalletProvider())
                .mobileNumber(recipient.getMobileNumber())
                .firstName(recipient.getFirstName())
                .lastName(recipient.getLastName())
                .createdAt(recipient.getCreatedAt())
                .build();
    }

    public Recipient toEntity(CreateRecipientRequest request, java.util.UUID userId) {
        return Recipient.builder()
                .userId(userId)
                .nickname(request.getNickname())
                .type(request.getType())
                .country(request.getCountry().toUpperCase())
                .currency(request.getCurrency().toUpperCase())
                .bankName(request.getBankName())
                .bankCode(request.getBankCode())
                .accountNumber(request.getAccountNumber())
                .accountName(request.getAccountName())
                .walletProvider(request.getWalletProvider())
                .mobileNumber(request.getMobileNumber())
                .firstName(request.getFirstName())
                .lastName(request.getLastName())
                .build();
    }
}
```

**Step 3: Commit**

```bash
git add src/main/java/com/movra/payment/mapper/
git commit -m "feat(payment): add DTO mappers for Transfer and Recipient"
```

---

### Task 5: Create RecipientRepository and RecipientService

**Files:**
- Create: `src/main/java/com/movra/payment/repository/RecipientRepository.java`
- Create: `src/main/java/com/movra/payment/service/RecipientService.java`

**Step 1: Create RecipientRepository**

```java
package com.movra.payment.repository;

import com.movra.payment.model.Recipient;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;
import java.util.UUID;

@Repository
public interface RecipientRepository extends JpaRepository<Recipient, UUID> {

    Page<Recipient> findByUserId(UUID userId, Pageable pageable);

    Optional<Recipient> findByIdAndUserId(UUID id, UUID userId);

    boolean existsByIdAndUserId(UUID id, UUID userId);
}
```

**Step 2: Create RecipientService**

```java
package com.movra.payment.service;

import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.exception.ResourceNotFoundException;
import com.movra.payment.mapper.RecipientMapper;
import com.movra.payment.model.Recipient;
import com.movra.payment.repository.RecipientRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class RecipientService {

    private final RecipientRepository recipientRepository;
    private final RecipientMapper recipientMapper;

    @Transactional
    public Recipient createRecipient(UUID userId, CreateRecipientRequest request) {
        Recipient recipient = recipientMapper.toEntity(request, userId);
        recipient = recipientRepository.save(recipient);
        log.info("Created recipient: {} for user: {}", recipient.getId(), userId);
        return recipient;
    }

    public Recipient getRecipient(UUID recipientId, UUID userId) {
        return recipientRepository.findByIdAndUserId(recipientId, userId)
                .orElseThrow(() -> new ResourceNotFoundException("Recipient", recipientId.toString()));
    }

    public Page<Recipient> listRecipients(UUID userId, Pageable pageable) {
        return recipientRepository.findByUserId(userId, pageable);
    }

    @Transactional
    public void deleteRecipient(UUID recipientId, UUID userId) {
        Recipient recipient = getRecipient(recipientId, userId);
        recipientRepository.delete(recipient);
        log.info("Deleted recipient: {} for user: {}", recipientId, userId);
    }

    public boolean existsForUser(UUID recipientId, UUID userId) {
        return recipientRepository.existsByIdAndUserId(recipientId, userId);
    }
}
```

**Step 3: Commit**

```bash
git add src/main/java/com/movra/payment/repository/RecipientRepository.java
git add src/main/java/com/movra/payment/service/RecipientService.java
git commit -m "feat(payment): add RecipientRepository and RecipientService"
```

---

### Task 6: Create REST Controllers

**Files:**
- Create: `src/main/java/com/movra/payment/controller/TransferController.java`
- Create: `src/main/java/com/movra/payment/controller/RecipientController.java`
- Create: `src/main/java/com/movra/payment/controller/HealthController.java`

**Step 1: Create HealthController**

```java
package com.movra.payment.controller;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
public class HealthController {

    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of(
                "status", "healthy",
                "service", "payment-service"
        ));
    }

    @GetMapping("/ready")
    public ResponseEntity<Map<String, String>> ready() {
        return ResponseEntity.ok(Map.of(
                "status", "ready",
                "service", "payment-service"
        ));
    }
}
```

**Step 2: Create RecipientController**

```java
package com.movra.payment.controller;

import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.dto.RecipientResponse;
import com.movra.payment.mapper.RecipientMapper;
import com.movra.payment.service.RecipientService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.web.PageableDefault;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/recipients")
@RequiredArgsConstructor
public class RecipientController {

    private final RecipientService recipientService;
    private final RecipientMapper recipientMapper;

    @PostMapping
    public ResponseEntity<RecipientResponse> createRecipient(
            @RequestHeader("X-User-Id") UUID userId,
            @Valid @RequestBody CreateRecipientRequest request) {

        var recipient = recipientService.createRecipient(userId, request);
        return ResponseEntity.status(HttpStatus.CREATED)
                .body(recipientMapper.toResponse(recipient));
    }

    @GetMapping("/{recipientId}")
    public ResponseEntity<RecipientResponse> getRecipient(
            @RequestHeader("X-User-Id") UUID userId,
            @PathVariable UUID recipientId) {

        var recipient = recipientService.getRecipient(recipientId, userId);
        return ResponseEntity.ok(recipientMapper.toResponse(recipient));
    }

    @GetMapping
    public ResponseEntity<Page<RecipientResponse>> listRecipients(
            @RequestHeader("X-User-Id") UUID userId,
            @PageableDefault(size = 20) Pageable pageable) {

        var recipients = recipientService.listRecipients(userId, pageable)
                .map(recipientMapper::toResponse);
        return ResponseEntity.ok(recipients);
    }

    @DeleteMapping("/{recipientId}")
    public ResponseEntity<Map<String, Boolean>> deleteRecipient(
            @RequestHeader("X-User-Id") UUID userId,
            @PathVariable UUID recipientId) {

        recipientService.deleteRecipient(recipientId, userId);
        return ResponseEntity.ok(Map.of("success", true));
    }
}
```

**Step 3: Create TransferController**

```java
package com.movra.payment.controller;

import com.movra.payment.dto.*;
import com.movra.payment.mapper.TransferMapper;
import com.movra.payment.model.TransferStatus;
import com.movra.payment.service.TransferService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.web.PageableDefault;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/transfers")
@RequiredArgsConstructor
public class TransferController {

    private final TransferService transferService;
    private final TransferMapper transferMapper;

    @PostMapping
    public ResponseEntity<TransferResponse> createTransfer(
            @RequestHeader("X-User-Id") UUID userId,
            @Valid @RequestBody CreateTransferRequest request) {

        var transfer = transferService.createTransfer(
                userId,
                request.getIdempotencyKey(),
                request.getSourceCurrency(),
                request.getSourceAmount(),
                request.getTargetCurrency(),
                request.getFundingMethod(),
                null, // PayoutMethod comes from recipient
                UUID.fromString(request.getRecipientId()),
                request.getRateLockId()
        );

        return ResponseEntity.status(HttpStatus.CREATED)
                .body(transferMapper.toResponse(transfer));
    }

    @GetMapping("/{transferId}")
    public ResponseEntity<TransferResponse> getTransfer(
            @RequestHeader("X-User-Id") UUID userId,
            @PathVariable UUID transferId) {

        var transfer = transferService.getTransfer(transferId, userId);
        return ResponseEntity.ok(transferMapper.toResponse(transfer));
    }

    @GetMapping
    public ResponseEntity<Page<TransferResponse>> listTransfers(
            @RequestHeader("X-User-Id") UUID userId,
            @RequestParam(required = false) TransferStatus status,
            @PageableDefault(size = 20) Pageable pageable) {

        var transfers = transferService.listTransfers(userId, status, pageable)
                .map(transferMapper::toResponse);
        return ResponseEntity.ok(transfers);
    }

    @PostMapping("/{transferId}/confirm")
    public ResponseEntity<TransferResponse> confirmTransfer(
            @RequestHeader("X-User-Id") UUID userId,
            @PathVariable UUID transferId) {

        var transfer = transferService.confirmTransfer(transferId, userId);
        return ResponseEntity.ok(transferMapper.toResponse(transfer));
    }

    @PostMapping("/{transferId}/cancel")
    public ResponseEntity<TransferResponse> cancelTransfer(
            @RequestHeader("X-User-Id") UUID userId,
            @PathVariable UUID transferId,
            @RequestBody(required = false) Map<String, String> body) {

        String reason = body != null ? body.get("reason") : "Cancelled by user";
        var transfer = transferService.cancelTransfer(transferId, userId, reason);
        return ResponseEntity.ok(transferMapper.toResponse(transfer));
    }

    @PostMapping("/quote")
    public ResponseEntity<QuoteResponse> getQuote(
            @RequestHeader("X-User-Id") UUID userId,
            @Valid @RequestBody QuoteRequest request) {

        var quote = transferService.getQuote(
                request.getSourceCurrency(),
                request.getSourceAmount(),
                request.getTargetCurrency(),
                request.getFundingMethod()
        );
        return ResponseEntity.ok(quote);
    }
}
```

**Step 4: Commit**

```bash
git add src/main/java/com/movra/payment/controller/
git commit -m "feat(payment): add REST controllers for transfers, recipients, health"
```

---

### Task 7: Update TransferService with Quote Method

**Files:**
- Modify: `src/main/java/com/movra/payment/service/TransferService.java`

**Step 1: Add getQuote method to TransferService**

Add this method to TransferService.java after the existing methods:

```java
public QuoteResponse getQuote(String sourceCurrency, BigDecimal sourceAmount,
                              String targetCurrency, FundingMethod fundingMethod) {
    // Get rate (mock for now, will integrate with Exchange Rate Service later)
    BigDecimal exchangeRate = getMockRate(sourceCurrency, targetCurrency);
    BigDecimal fee = calculateFee(sourceAmount, fundingMethod);
    BigDecimal netAmount = sourceAmount.subtract(fee);
    BigDecimal targetAmount = netAmount.multiply(exchangeRate).setScale(2, RoundingMode.HALF_UP);

    return QuoteResponse.builder()
            .sourceCurrency(sourceCurrency)
            .sourceAmount(sourceAmount)
            .targetCurrency(targetCurrency)
            .targetAmount(targetAmount)
            .exchangeRate(exchangeRate)
            .fee(fee)
            .totalCost(sourceAmount)
            .validUntil(Instant.now().plusSeconds(30))
            .build();
}
```

Also add the import:
```java
import com.movra.payment.dto.QuoteResponse;
```

**Step 2: Commit**

```bash
git add src/main/java/com/movra/payment/service/TransferService.java
git commit -m "feat(payment): add quote generation to TransferService"
```

---

## Phase B: gRPC Integration with Exchange Rate Service

### Task 8: Create gRPC Client for Exchange Rate Service

**Files:**
- Create: `src/main/java/com/movra/payment/grpc/ExchangeRateClient.java`

**Step 1: Create ExchangeRateClient**

```java
package com.movra.payment.grpc;

import lombok.extern.slf4j.Slf4j;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.springframework.stereotype.Component;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.Optional;

@Component
@Slf4j
public class ExchangeRateClient {

    // TODO: Uncomment when proto is regenerated with correct package
    // @GrpcClient("exchange-rate-service")
    // private ExchangeRateServiceGrpc.ExchangeRateServiceBlockingStub exchangeRateStub;

    public record RateInfo(
            BigDecimal midRate,
            BigDecimal buyRate,
            String rateLockId,
            Instant expiresAt
    ) {}

    public Optional<RateInfo> getRate(String sourceCurrency, String targetCurrency) {
        try {
            // TODO: Call actual gRPC service
            // var request = GetRateRequest.newBuilder()
            //         .setSourceCurrency(sourceCurrency)
            //         .setTargetCurrency(targetCurrency)
            //         .build();
            // var response = exchangeRateStub.getRate(request);

            // Mock response for now
            BigDecimal mockRate = getMockRate(sourceCurrency, targetCurrency);
            if (mockRate == null) {
                return Optional.empty();
            }

            return Optional.of(new RateInfo(
                    mockRate,
                    mockRate.multiply(new BigDecimal("0.997")), // 0.3% margin
                    null,
                    Instant.now().plusSeconds(30)
            ));
        } catch (Exception e) {
            log.error("Failed to get rate from Exchange Rate Service", e);
            return Optional.empty();
        }
    }

    public Optional<RateInfo> lockRate(String sourceCurrency, String targetCurrency, int durationSeconds) {
        try {
            // TODO: Call actual gRPC service
            BigDecimal mockRate = getMockRate(sourceCurrency, targetCurrency);
            if (mockRate == null) {
                return Optional.empty();
            }

            return Optional.of(new RateInfo(
                    mockRate,
                    mockRate.multiply(new BigDecimal("0.997")),
                    "lock_" + System.currentTimeMillis(),
                    Instant.now().plusSeconds(durationSeconds)
            ));
        } catch (Exception e) {
            log.error("Failed to lock rate", e);
            return Optional.empty();
        }
    }

    public Optional<RateInfo> getLockedRate(String rateLockId) {
        try {
            // TODO: Call actual gRPC service
            // For mock, we just return a rate if the lock ID looks valid
            if (rateLockId == null || !rateLockId.startsWith("lock_")) {
                return Optional.empty();
            }

            return Optional.of(new RateInfo(
                    new BigDecimal("42.50"),
                    new BigDecimal("42.37"),
                    rateLockId,
                    Instant.now().plusSeconds(30)
            ));
        } catch (Exception e) {
            log.error("Failed to get locked rate", e);
            return Optional.empty();
        }
    }

    private BigDecimal getMockRate(String from, String to) {
        if (from.equals("SGD") && to.equals("PHP")) return new BigDecimal("42.50");
        if (from.equals("SGD") && to.equals("INR")) return new BigDecimal("62.50");
        if (from.equals("SGD") && to.equals("IDR")) return new BigDecimal("11800");
        if (from.equals("USD") && to.equals("PHP")) return new BigDecimal("57.05");
        if (from.equals("SGD") && to.equals("USD")) return new BigDecimal("0.745");
        return null;
    }
}
```

**Step 2: Commit**

```bash
git add src/main/java/com/movra/payment/grpc/
git commit -m "feat(payment): add gRPC client for Exchange Rate Service"
```

---

### Task 9: Integrate Exchange Rate Client into TransferService

**Files:**
- Modify: `src/main/java/com/movra/payment/service/TransferService.java`

**Step 1: Inject ExchangeRateClient and use it**

Update TransferService constructor and fields:

```java
private final TransferRepository transferRepository;
private final TransferEventPublisher eventPublisher;
private final RecipientService recipientService;
private final ExchangeRateClient exchangeRateClient;
```

Update createTransfer to use real rates:

```java
@Transactional
public Transfer createTransfer(
        UUID userId,
        String idempotencyKey,
        String sourceCurrency,
        BigDecimal sourceAmount,
        String targetCurrency,
        FundingMethod fundingMethod,
        PayoutMethod payoutMethod,
        UUID recipientId,
        String rateLockId
) {
    // Check idempotency
    var existing = transferRepository.findByIdempotencyKey(idempotencyKey);
    if (existing.isPresent()) {
        log.info("Returning existing transfer for idempotency key: {}", idempotencyKey);
        return existing.get();
    }

    // Validate recipient exists and belongs to user
    var recipient = recipientService.getRecipient(recipientId, userId);
    PayoutMethod actualPayoutMethod = payoutMethod != null ? payoutMethod : recipient.getType();

    // Get rate from Exchange Rate Service
    ExchangeRateClient.RateInfo rateInfo;
    if (rateLockId != null) {
        rateInfo = exchangeRateClient.getLockedRate(rateLockId)
                .orElseThrow(() -> new RateLockExpiredException(rateLockId));
    } else {
        rateInfo = exchangeRateClient.lockRate(sourceCurrency, targetCurrency, 30)
                .orElseThrow(() -> new IllegalArgumentException(
                        "Unsupported currency pair: " + sourceCurrency + "/" + targetCurrency));
    }

    BigDecimal fee = calculateFee(sourceAmount, fundingMethod);
    BigDecimal netAmount = sourceAmount.subtract(fee);
    BigDecimal targetAmount = netAmount.multiply(rateInfo.buyRate()).setScale(2, RoundingMode.HALF_UP);

    var transfer = Transfer.builder()
            .userId(userId)
            .idempotencyKey(idempotencyKey)
            .status(TransferStatus.CREATED)
            .sourceCurrency(sourceCurrency)
            .sourceAmount(sourceAmount)
            .targetCurrency(targetCurrency)
            .targetAmount(targetAmount)
            .exchangeRate(rateInfo.buyRate())
            .rateLockId(rateInfo.rateLockId())
            .rateExpiresAt(rateInfo.expiresAt())
            .feeAmount(fee)
            .feeCurrency(sourceCurrency)
            .fundingMethod(fundingMethod)
            .payoutMethod(actualPayoutMethod)
            .recipientId(recipientId)
            .build();

    // Generate funding reference for bank transfer
    if (fundingMethod == FundingMethod.BANK_TRANSFER) {
        transfer.setFundingBankName("DBS Bank");
        transfer.setFundingAccountNumber("1234567890");
        transfer.setFundingAccountName("Movra Pte Ltd");
        transfer.setFundingReference("MOV" + System.currentTimeMillis());
    }

    transfer = transferRepository.save(transfer);
    log.info("Created transfer: {} for user: {}", transfer.getId(), userId);

    // Publish event
    eventPublisher.publishTransferInitiated(transfer);

    return transfer;
}
```

Add the import:
```java
import com.movra.payment.grpc.ExchangeRateClient;
import com.movra.payment.exception.RateLockExpiredException;
```

**Step 2: Commit**

```bash
git add src/main/java/com/movra/payment/service/TransferService.java
git commit -m "feat(payment): integrate Exchange Rate gRPC client"
```

---

## Phase C: Testing

### Task 10: Create Unit Tests for TransferService

**Files:**
- Create: `src/test/java/com/movra/payment/service/TransferServiceTest.java`

**Step 1: Create TransferServiceTest**

```java
package com.movra.payment.service;

import com.movra.payment.grpc.ExchangeRateClient;
import com.movra.payment.kafka.TransferEventPublisher;
import com.movra.payment.model.*;
import com.movra.payment.repository.TransferRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.math.BigDecimal;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class TransferServiceTest {

    @Mock
    private TransferRepository transferRepository;

    @Mock
    private TransferEventPublisher eventPublisher;

    @Mock
    private RecipientService recipientService;

    @Mock
    private ExchangeRateClient exchangeRateClient;

    private TransferService transferService;

    @BeforeEach
    void setUp() {
        transferService = new TransferService(
                transferRepository,
                eventPublisher,
                recipientService,
                exchangeRateClient
        );
    }

    @Test
    void createTransfer_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        UUID recipientId = UUID.randomUUID();
        String idempotencyKey = "test-key-123";

        Recipient recipient = Recipient.builder()
                .id(recipientId)
                .userId(userId)
                .type(PayoutMethod.BANK_ACCOUNT)
                .build();

        when(transferRepository.findByIdempotencyKey(idempotencyKey))
                .thenReturn(Optional.empty());
        when(recipientService.getRecipient(recipientId, userId))
                .thenReturn(recipient);
        when(exchangeRateClient.lockRate(eq("SGD"), eq("PHP"), anyInt()))
                .thenReturn(Optional.of(new ExchangeRateClient.RateInfo(
                        new BigDecimal("42.50"),
                        new BigDecimal("42.37"),
                        "lock_123",
                        Instant.now().plusSeconds(30)
                )));
        when(transferRepository.save(any(Transfer.class)))
                .thenAnswer(inv -> {
                    Transfer t = inv.getArgument(0);
                    t.setId(UUID.randomUUID());
                    return t;
                });

        // When
        Transfer result = transferService.createTransfer(
                userId,
                idempotencyKey,
                "SGD",
                new BigDecimal("100.00"),
                "PHP",
                FundingMethod.BANK_TRANSFER,
                null,
                recipientId,
                null
        );

        // Then
        assertThat(result).isNotNull();
        assertThat(result.getStatus()).isEqualTo(TransferStatus.CREATED);
        assertThat(result.getSourceCurrency()).isEqualTo("SGD");
        assertThat(result.getTargetCurrency()).isEqualTo("PHP");
        assertThat(result.getRateLockId()).isEqualTo("lock_123");

        verify(eventPublisher).publishTransferInitiated(result);
    }

    @Test
    void createTransfer_Idempotency_ReturnsExisting() {
        // Given
        UUID userId = UUID.randomUUID();
        String idempotencyKey = "existing-key";
        Transfer existingTransfer = Transfer.builder()
                .id(UUID.randomUUID())
                .idempotencyKey(idempotencyKey)
                .status(TransferStatus.CREATED)
                .build();

        when(transferRepository.findByIdempotencyKey(idempotencyKey))
                .thenReturn(Optional.of(existingTransfer));

        // When
        Transfer result = transferService.createTransfer(
                userId, idempotencyKey, "SGD", BigDecimal.TEN, "PHP",
                FundingMethod.BANK_TRANSFER, null, UUID.randomUUID(), null
        );

        // Then
        assertThat(result).isEqualTo(existingTransfer);
        verify(transferRepository, never()).save(any());
        verify(eventPublisher, never()).publishTransferInitiated(any());
    }

    @Test
    void confirmTransfer_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        UUID transferId = UUID.randomUUID();
        Transfer transfer = Transfer.builder()
                .id(transferId)
                .userId(userId)
                .status(TransferStatus.CREATED)
                .rateExpiresAt(Instant.now().plusSeconds(30))
                .build();

        when(transferRepository.findById(transferId))
                .thenReturn(Optional.of(transfer));
        when(transferRepository.save(any(Transfer.class)))
                .thenAnswer(inv -> inv.getArgument(0));

        // When
        Transfer result = transferService.confirmTransfer(transferId, userId);

        // Then
        assertThat(result.getStatus()).isEqualTo(TransferStatus.AWAITING_FUNDS);
    }

    @Test
    void confirmTransfer_RateExpired_ThrowsException() {
        // Given
        UUID userId = UUID.randomUUID();
        UUID transferId = UUID.randomUUID();
        Transfer transfer = Transfer.builder()
                .id(transferId)
                .userId(userId)
                .status(TransferStatus.CREATED)
                .rateExpiresAt(Instant.now().minusSeconds(10)) // Expired
                .build();

        when(transferRepository.findById(transferId))
                .thenReturn(Optional.of(transfer));

        // When/Then
        assertThatThrownBy(() -> transferService.confirmTransfer(transferId, userId))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Rate lock has expired");
    }

    @Test
    void cancelTransfer_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        UUID transferId = UUID.randomUUID();
        Transfer transfer = Transfer.builder()
                .id(transferId)
                .userId(userId)
                .status(TransferStatus.CREATED)
                .build();

        when(transferRepository.findById(transferId))
                .thenReturn(Optional.of(transfer));
        when(transferRepository.save(any(Transfer.class)))
                .thenAnswer(inv -> inv.getArgument(0));

        // When
        Transfer result = transferService.cancelTransfer(transferId, userId, "Changed mind");

        // Then
        assertThat(result.getStatus()).isEqualTo(TransferStatus.CANCELLED);
        assertThat(result.getFailureReason()).isEqualTo("Changed mind");
    }
}
```

**Step 2: Run tests**

```bash
./mvnw test -Dtest=TransferServiceTest
```

Expected: All tests pass

**Step 3: Commit**

```bash
git add src/test/java/com/movra/payment/service/TransferServiceTest.java
git commit -m "test(payment): add TransferService unit tests"
```

---

### Task 11: Create Integration Tests

**Files:**
- Create: `src/test/java/com/movra/payment/controller/TransferControllerIntegrationTest.java`

**Step 1: Add Testcontainers dependencies to pom.xml**

Add to pom.xml:
```xml
<dependency>
    <groupId>org.testcontainers</groupId>
    <artifactId>junit-jupiter</artifactId>
    <scope>test</scope>
</dependency>
<dependency>
    <groupId>org.testcontainers</groupId>
    <artifactId>postgresql</artifactId>
    <scope>test</scope>
</dependency>
```

**Step 2: Create integration test**

```java
package com.movra.payment.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.dto.CreateTransferRequest;
import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.math.BigDecimal;
import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest
@AutoConfigureMockMvc
@Testcontainers
class TransferControllerIntegrationTest {

    @Container
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:15-alpine");

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
        registry.add("spring.kafka.bootstrap-servers", () -> "localhost:9092");
    }

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    private static final UUID TEST_USER_ID = UUID.randomUUID();

    @Test
    void healthCheck() throws Exception {
        mockMvc.perform(get("/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("healthy"));
    }

    @Test
    void createRecipient_Success() throws Exception {
        var request = new CreateRecipientRequest();
        request.setNickname("John's Account");
        request.setType(PayoutMethod.BANK_ACCOUNT);
        request.setCountry("PH");
        request.setCurrency("PHP");
        request.setBankName("BDO");
        request.setAccountNumber("1234567890");
        request.setAccountName("John Doe");

        mockMvc.perform(post("/api/recipients")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.nickname").value("John's Account"))
                .andExpect(jsonPath("$.type").value("BANK_ACCOUNT"));
    }

    @Test
    void createTransfer_ValidationError() throws Exception {
        var request = new CreateTransferRequest();
        // Missing required fields

        mockMvc.perform(post("/api/transfers")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("VALIDATION_FAILED"));
    }

    @Test
    void getQuote_Success() throws Exception {
        var request = """
                {
                    "sourceCurrency": "SGD",
                    "sourceAmount": 100.00,
                    "targetCurrency": "PHP",
                    "fundingMethod": "BANK_TRANSFER"
                }
                """;

        mockMvc.perform(post("/api/transfers/quote")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(request))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.sourceCurrency").value("SGD"))
                .andExpect(jsonPath("$.targetCurrency").value("PHP"))
                .andExpect(jsonPath("$.exchangeRate").exists())
                .andExpect(jsonPath("$.fee").exists());
    }
}
```

**Step 3: Run integration tests**

```bash
./mvnw test -Dtest=TransferControllerIntegrationTest
```

**Step 4: Commit**

```bash
git add pom.xml
git add src/test/java/com/movra/payment/controller/TransferControllerIntegrationTest.java
git commit -m "test(payment): add integration tests with Testcontainers"
```

---

## Final Summary

After completing all tasks, the Payment Service will have:

1. **DTOs** - CreateTransferRequest, TransferResponse, QuoteRequest/Response, RecipientDTOs
2. **Exception Handling** - GlobalExceptionHandler with consistent API errors
3. **Mappers** - TransferMapper, RecipientMapper
4. **REST Controllers** - TransferController, RecipientController, HealthController
5. **gRPC Client** - ExchangeRateClient (mock for now, ready for real integration)
6. **Unit Tests** - TransferServiceTest
7. **Integration Tests** - TransferControllerIntegrationTest with Testcontainers

**Endpoints available:**
- `POST /api/transfers` - Create transfer
- `GET /api/transfers/{id}` - Get transfer
- `GET /api/transfers` - List transfers
- `POST /api/transfers/{id}/confirm` - Confirm transfer
- `POST /api/transfers/{id}/cancel` - Cancel transfer
- `POST /api/transfers/quote` - Get quote
- `POST /api/recipients` - Create recipient
- `GET /api/recipients/{id}` - Get recipient
- `GET /api/recipients` - List recipients
- `DELETE /api/recipients/{id}` - Delete recipient
- `GET /health` - Health check
- `GET /ready` - Readiness check
