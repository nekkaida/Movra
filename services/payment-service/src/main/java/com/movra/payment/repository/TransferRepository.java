package com.movra.payment.repository;

import com.movra.payment.model.Transfer;
import com.movra.payment.model.TransferStatus;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;
import java.util.UUID;

@Repository
public interface TransferRepository extends JpaRepository<Transfer, UUID> {

    Optional<Transfer> findByIdempotencyKey(String idempotencyKey);

    Page<Transfer> findByUserId(UUID userId, Pageable pageable);

    Page<Transfer> findByUserIdAndStatus(UUID userId, TransferStatus status, Pageable pageable);

    boolean existsByIdempotencyKey(String idempotencyKey);
}
