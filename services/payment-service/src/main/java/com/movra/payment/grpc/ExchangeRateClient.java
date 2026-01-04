package com.movra.payment.grpc;

import lombok.extern.slf4j.Slf4j;
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
