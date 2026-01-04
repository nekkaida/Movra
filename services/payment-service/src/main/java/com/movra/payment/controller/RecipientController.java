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
