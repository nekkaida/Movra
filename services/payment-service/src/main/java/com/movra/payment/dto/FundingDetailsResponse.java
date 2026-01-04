package com.movra.payment.dto;

import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class FundingDetailsResponse {
    private String bankName;
    private String accountNumber;
    private String accountName;
    private String reference;
}
