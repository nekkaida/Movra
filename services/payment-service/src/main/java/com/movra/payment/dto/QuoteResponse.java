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
