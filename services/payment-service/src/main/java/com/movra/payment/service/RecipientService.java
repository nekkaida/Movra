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
