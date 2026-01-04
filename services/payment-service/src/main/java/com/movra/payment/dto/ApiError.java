package com.movra.payment.dto;

import lombok.Builder;
import lombok.Data;

import java.time.Instant;
import java.util.Map;

@Data
@Builder
public class ApiError {
    private String code;
    private String message;
    private Map<String, String> details;
    private Instant timestamp;
    private String path;
}
