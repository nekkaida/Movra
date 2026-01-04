package com.movra.payment.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.movra.payment.dto.CreateRecipientRequest;
import com.movra.payment.dto.CreateTransferRequest;
import com.movra.payment.model.FundingMethod;
import com.movra.payment.model.PayoutMethod;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.math.BigDecimal;
import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest
@AutoConfigureMockMvc
@Testcontainers
class TransferControllerIntegrationTest {

    @Container
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:15-alpine");

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
        registry.add("spring.kafka.bootstrap-servers", () -> "localhost:9092");
        registry.add("spring.kafka.producer.properties.max.block.ms", () -> "100");
    }

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    private static final UUID TEST_USER_ID = UUID.randomUUID();

    @Test
    void healthCheck() throws Exception {
        mockMvc.perform(get("/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("healthy"));
    }

    @Test
    void createRecipient_Success() throws Exception {
        var request = new CreateRecipientRequest();
        request.setNickname("John's Account");
        request.setType(PayoutMethod.BANK_ACCOUNT);
        request.setCountry("PH");
        request.setCurrency("PHP");
        request.setBankName("BDO");
        request.setAccountNumber("1234567890");
        request.setAccountName("John Doe");

        mockMvc.perform(post("/api/recipients")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.nickname").value("John's Account"))
                .andExpect(jsonPath("$.type").value("BANK_ACCOUNT"));
    }

    @Test
    void createTransfer_ValidationError() throws Exception {
        var request = new CreateTransferRequest();
        // Missing required fields

        mockMvc.perform(post("/api/transfers")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("VALIDATION_FAILED"));
    }

    @Test
    void getQuote_Success() throws Exception {
        var request = """
                {
                    "sourceCurrency": "SGD",
                    "sourceAmount": 100.00,
                    "targetCurrency": "PHP",
                    "fundingMethod": "BANK_TRANSFER"
                }
                """;

        mockMvc.perform(post("/api/transfers/quote")
                        .header("X-User-Id", TEST_USER_ID.toString())
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(request))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.sourceCurrency").value("SGD"))
                .andExpect(jsonPath("$.targetCurrency").value("PHP"))
                .andExpect(jsonPath("$.exchangeRate").exists())
                .andExpect(jsonPath("$.fee").exists());
    }

    @Test
    void listRecipients_Empty() throws Exception {
        UUID newUserId = UUID.randomUUID();

        mockMvc.perform(get("/api/recipients")
                        .header("X-User-Id", newUserId.toString()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.content").isArray())
                .andExpect(jsonPath("$.content").isEmpty());
    }

    @Test
    void getRecipient_NotFound() throws Exception {
        UUID randomRecipientId = UUID.randomUUID();

        mockMvc.perform(get("/api/recipients/" + randomRecipientId)
                        .header("X-User-Id", TEST_USER_ID.toString()))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("RESOURCE_NOT_FOUND"));
    }
}
