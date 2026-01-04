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

    @Test
    void getQuote_Success() {
        // When
        var quote = transferService.getQuote("SGD", new BigDecimal("100.00"), "PHP", FundingMethod.BANK_TRANSFER);

        // Then
        assertThat(quote.getSourceCurrency()).isEqualTo("SGD");
        assertThat(quote.getTargetCurrency()).isEqualTo("PHP");
        assertThat(quote.getExchangeRate()).isNotNull();
        assertThat(quote.getFee()).isNotNull();
        assertThat(quote.getValidUntil()).isAfter(Instant.now());
    }
}
