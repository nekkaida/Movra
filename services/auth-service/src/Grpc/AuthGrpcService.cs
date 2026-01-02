using Grpc.Core;
using AuthService.Services;
using AuthService.Models;

namespace AuthService.Grpc;

public class AuthGrpcService : Movra.Auth.AuthService.AuthServiceBase
{
    private readonly IUserService _userService;
    private readonly ITokenService _tokenService;
    private readonly ILogger<AuthGrpcService> _logger;

    public AuthGrpcService(IUserService userService, ITokenService tokenService, ILogger<AuthGrpcService> logger)
    {
        _userService = userService;
        _tokenService = tokenService;
        _logger = logger;
    }

    public override async Task<Movra.Auth.VerifyTokenResponse> VerifyToken(
        Movra.Auth.VerifyTokenRequest request,
        ServerCallContext context)
    {
        var principal = _tokenService.ValidateAccessToken(request.Token);

        if (principal == null)
        {
            return new Movra.Auth.VerifyTokenResponse
            {
                Valid = false,
                Error = new Movra.Common.Error { Code = "INVALID_TOKEN", Message = "Token is invalid or expired" }
            };
        }

        var userId = principal.Claims.FirstOrDefault(c => c.Type == "sub")?.Value;
        var kycLevelStr = principal.Claims.FirstOrDefault(c => c.Type == "kyc_level")?.Value;
        var kycLevel = Enum.TryParse<KycLevel>(kycLevelStr, out var level) ? level : KycLevel.None;

        return new Movra.Auth.VerifyTokenResponse
        {
            Valid = true,
            UserId = userId ?? "",
            KycLevel = MapKycLevel(kycLevel)
        };
    }

    public override async Task<Movra.Auth.GetUserKYCLevelResponse> GetUserKYCLevel(
        Movra.Auth.GetUserKYCLevelRequest request,
        ServerCallContext context)
    {
        if (!Guid.TryParse(request.UserId, out var userId))
        {
            return new Movra.Auth.GetUserKYCLevelResponse
            {
                Error = new Movra.Common.Error { Code = "INVALID_USER_ID", Message = "Invalid user ID format" }
            };
        }

        var user = await _userService.GetByIdAsync(userId);
        if (user == null)
        {
            return new Movra.Auth.GetUserKYCLevelResponse
            {
                Error = new Movra.Common.Error { Code = "USER_NOT_FOUND", Message = "User not found" }
            };
        }

        var limits = await _userService.GetTransferLimitsAsync(user.KycLevel);

        return new Movra.Auth.GetUserKYCLevelResponse
        {
            UserId = user.Id.ToString(),
            KycLevel = MapKycLevel(user.KycLevel),
            Limits = new Movra.Auth.TransferLimits
            {
                DailyLimit = new Movra.Common.Money
                {
                    Currency = limits.Currency,
                    Amount = limits.DailyLimit.ToString("F2")
                },
                MonthlyLimit = new Movra.Common.Money
                {
                    Currency = limits.Currency,
                    Amount = limits.MonthlyLimit.ToString("F2")
                },
                PerTransactionLimit = new Movra.Common.Money
                {
                    Currency = limits.Currency,
                    Amount = limits.PerTransactionLimit.ToString("F2")
                }
            }
        };
    }

    public override async Task<Movra.Auth.GetUserResponse> GetUser(
        Movra.Auth.GetUserRequest request,
        ServerCallContext context)
    {
        if (!Guid.TryParse(request.UserId, out var userId))
        {
            return new Movra.Auth.GetUserResponse
            {
                Error = new Movra.Common.Error { Code = "INVALID_USER_ID", Message = "Invalid user ID format" }
            };
        }

        var user = await _userService.GetByIdAsync(userId);
        if (user == null)
        {
            return new Movra.Auth.GetUserResponse
            {
                Error = new Movra.Common.Error { Code = "USER_NOT_FOUND", Message = "User not found" }
            };
        }

        return new Movra.Auth.GetUserResponse
        {
            User = MapUser(user)
        };
    }

    public override async Task<Movra.Auth.RegisterResponse> Register(
        Movra.Auth.RegisterRequest request,
        ServerCallContext context)
    {
        try
        {
            var user = await _userService.CreateAsync(
                request.Email,
                request.Password,
                request.FirstName,
                request.LastName,
                request.Phone
            );

            var accessToken = _tokenService.GenerateAccessToken(user);
            var refreshToken = await _tokenService.CreateRefreshTokenAsync(user);

            _logger.LogInformation("User registered via gRPC: {UserId}", user.Id);

            return new Movra.Auth.RegisterResponse
            {
                User = MapUser(user),
                AccessToken = accessToken,
                RefreshToken = refreshToken.Token
            };
        }
        catch (InvalidOperationException ex)
        {
            return new Movra.Auth.RegisterResponse
            {
                Error = new Movra.Common.Error { Code = "REGISTRATION_FAILED", Message = ex.Message }
            };
        }
    }

    public override async Task<Movra.Auth.LoginResponse> Login(
        Movra.Auth.LoginRequest request,
        ServerCallContext context)
    {
        var user = await _userService.GetByEmailAsync(request.Email);
        if (user == null || !await _userService.ValidatePasswordAsync(user, request.Password))
        {
            return new Movra.Auth.LoginResponse
            {
                Error = new Movra.Common.Error { Code = "INVALID_CREDENTIALS", Message = "Invalid email or password" }
            };
        }

        var accessToken = _tokenService.GenerateAccessToken(user);
        var refreshToken = await _tokenService.CreateRefreshTokenAsync(user);

        _logger.LogInformation("User logged in via gRPC: {UserId}", user.Id);

        return new Movra.Auth.LoginResponse
        {
            User = MapUser(user),
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token,
            RequiresMfa = user.MfaEnabled
        };
    }

    public override async Task<Movra.Auth.RefreshTokenResponse> RefreshToken(
        Movra.Auth.RefreshTokenRequest request,
        ServerCallContext context)
    {
        var storedToken = await _tokenService.GetRefreshTokenAsync(request.RefreshToken);
        if (storedToken == null || !storedToken.IsActive)
        {
            return new Movra.Auth.RefreshTokenResponse
            {
                Error = new Movra.Common.Error { Code = "INVALID_TOKEN", Message = "Invalid or expired refresh token" }
            };
        }

        // Revoke old token and create new ones
        await _tokenService.RevokeRefreshTokenAsync(request.RefreshToken);

        var accessToken = _tokenService.GenerateAccessToken(storedToken.User);
        var newRefreshToken = await _tokenService.CreateRefreshTokenAsync(storedToken.User);

        return new Movra.Auth.RefreshTokenResponse
        {
            AccessToken = accessToken,
            RefreshToken = newRefreshToken.Token,
            ExpiresAt = new Movra.Common.Timestamp
            {
                Seconds = new DateTimeOffset(newRefreshToken.ExpiresAt).ToUnixTimeSeconds()
            }
        };
    }

    private static Movra.Auth.User MapUser(User user)
    {
        return new Movra.Auth.User
        {
            Id = user.Id.ToString(),
            Email = user.Email,
            Phone = user.Phone ?? "",
            FirstName = user.FirstName,
            LastName = user.LastName,
            KycLevel = MapKycLevel(user.KycLevel),
            CreatedAt = new Movra.Common.Timestamp
            {
                Seconds = new DateTimeOffset(user.CreatedAt).ToUnixTimeSeconds()
            },
            UpdatedAt = new Movra.Common.Timestamp
            {
                Seconds = new DateTimeOffset(user.UpdatedAt).ToUnixTimeSeconds()
            }
        };
    }

    private static Movra.Auth.KYCLevel MapKycLevel(KycLevel level)
    {
        return level switch
        {
            KycLevel.None => Movra.Auth.KYCLevel.None,
            KycLevel.Basic => Movra.Auth.KYCLevel.Basic,
            KycLevel.Verified => Movra.Auth.KYCLevel.Verified,
            KycLevel.Premium => Movra.Auth.KYCLevel.Premium,
            _ => Movra.Auth.KYCLevel.None
        };
    }
}
