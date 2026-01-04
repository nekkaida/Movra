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
