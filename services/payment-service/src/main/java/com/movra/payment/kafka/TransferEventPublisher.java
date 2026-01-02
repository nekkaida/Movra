package com.movra.payment.kafka;

import com.movra.payment.model.Transfer;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;

@Component
@RequiredArgsConstructor
@Slf4j
public class TransferEventPublisher {

    private final KafkaTemplate<String, Object> kafkaTemplate;

    private static final String TOPIC_TRANSFER_INITIATED = "movra.transfers.initiated";
    private static final String TOPIC_FUNDS_RECEIVED = "movra.transfers.funds-received";
    private static final String TOPIC_TRANSFER_COMPLETED = "movra.transfers.completed";
    private static final String TOPIC_TRANSFER_FAILED = "movra.transfers.failed";

    public void publishTransferInitiated(Transfer transfer) {
        var event = buildEvent(transfer, "TRANSFER_INITIATED");
        kafkaTemplate.send(TOPIC_TRANSFER_INITIATED, transfer.getId().toString(), event);
        log.info("Published TRANSFER_INITIATED event for transfer: {}", transfer.getId());
    }

    public void publishFundsReceived(Transfer transfer) {
        var event = buildEvent(transfer, "FUNDS_RECEIVED");
        kafkaTemplate.send(TOPIC_FUNDS_RECEIVED, transfer.getId().toString(), event);
        log.info("Published FUNDS_RECEIVED event for transfer: {}", transfer.getId());
    }

    public void publishTransferCompleted(Transfer transfer) {
        var event = buildEvent(transfer, "TRANSFER_COMPLETED");
        kafkaTemplate.send(TOPIC_TRANSFER_COMPLETED, transfer.getId().toString(), event);
        log.info("Published TRANSFER_COMPLETED event for transfer: {}", transfer.getId());
    }

    public void publishTransferFailed(Transfer transfer, String reason) {
        var event = buildEvent(transfer, "TRANSFER_FAILED");
        event.put("failureReason", reason);
        kafkaTemplate.send(TOPIC_TRANSFER_FAILED, transfer.getId().toString(), event);
        log.info("Published TRANSFER_FAILED event for transfer: {} - reason: {}", transfer.getId(), reason);
    }

    private Map<String, Object> buildEvent(Transfer transfer, String eventType) {
        Map<String, Object> event = new HashMap<>();
        event.put("eventType", eventType);
        event.put("transferId", transfer.getId().toString());
        event.put("userId", transfer.getUserId().toString());
        event.put("status", transfer.getStatus().toString());
        event.put("sourceCurrency", transfer.getSourceCurrency());
        event.put("sourceAmount", transfer.getSourceAmount().toString());
        event.put("targetCurrency", transfer.getTargetCurrency());
        event.put("targetAmount", transfer.getTargetAmount().toString());
        event.put("timestamp", Instant.now().toString());
        return event;
    }
}
