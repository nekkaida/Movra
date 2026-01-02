package com.movra.payment.model;

import jakarta.persistence.*;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import lombok.Builder;

import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "recipients")
@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Recipient {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @Column(nullable = false)
    private UUID userId;

    @Column(nullable = false)
    private String nickname;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private PayoutMethod type;

    @Column(nullable = false, length = 2)
    private String country;

    @Column(nullable = false, length = 3)
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

    @Column(nullable = false, updatable = false)
    private Instant createdAt;

    @PrePersist
    protected void onCreate() {
        createdAt = Instant.now();
    }
}
