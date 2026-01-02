package com.movra.payment.model;

import jakarta.persistence.*;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import lombok.Builder;

import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "idempotency_keys")
@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class IdempotencyKey {

    @Id
    private String key;

    @Column(nullable = false)
    private UUID transferId;

    @Column(nullable = false)
    private UUID userId;

    @Column(columnDefinition = "TEXT")
    private String responseBody;

    private Integer responseStatus;

    @Column(nullable = false, updatable = false)
    private Instant createdAt;

    @Column(nullable = false)
    private Instant expiresAt;

    @PrePersist
    protected void onCreate() {
        createdAt = Instant.now();
        if (expiresAt == null) {
            expiresAt = createdAt.plusSeconds(86400); // 24 hours
        }
    }

    public boolean isExpired() {
        return Instant.now().isAfter(expiresAt);
    }
}
