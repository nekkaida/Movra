using Microsoft.AspNetCore.Mvc;
using AuthService.Dtos;
using AuthService.Services;
using AuthService.Exceptions;

namespace AuthService.Controllers;

[ApiController]
[Route("api/auth")]
public class AuthController : ControllerBase
{
    private readonly IUserService _userService;
    private readonly ITokenService _tokenService;
    private readonly ILogger<AuthController> _logger;
    private readonly IConfiguration _configuration;

    public AuthController(
        IUserService userService,
        ITokenService tokenService,
        ILogger<AuthController> logger,
        IConfiguration configuration)
    {
        _userService = userService;
        _tokenService = tokenService;
        _logger = logger;
        _configuration = configuration;
    }

    [HttpPost("register")]
    [ProducesResponseType(typeof(AuthResponse), StatusCodes.Status201Created)]
    [ProducesResponseType(typeof(ApiError), StatusCodes.Status400BadRequest)]
    [ProducesResponseType(typeof(ApiError), StatusCodes.Status409Conflict)]
    public async Task<IActionResult> Register([FromBody] RegisterRequest request)
    {
        var existingUser = await _userService.GetByEmailAsync(request.Email);
        if (existingUser != null)
        {
            throw new UserAlreadyExistsException(request.Email);
        }

        var user = await _userService.CreateAsync(
            request.Email,
            request.Password,
            request.FirstName,
            request.LastName,
            request.Phone
        );

        var accessToken = _tokenService.GenerateAccessToken(user);
        var refreshToken = await _tokenService.CreateRefreshTokenAsync(user);

        var expiryMinutes = int.Parse(_configuration["Jwt:ExpiryMinutes"] ?? "60");

        _logger.LogInformation("User registered: {UserId}", user.Id);

        var response = new AuthResponse
        {
            User = MapUserResponse(user),
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token,
            ExpiresAt = DateTime.UtcNow.AddMinutes(expiryMinutes)
        };

        return CreatedAtAction(nameof(GetProfile), new { }, response);
    }

    [HttpPost("login")]
    [ProducesResponseType(typeof(AuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiError), StatusCodes.Status401Unauthorized)]
    public async Task<IActionResult> Login([FromBody] LoginRequest request)
    {
        var user = await _userService.GetByEmailAsync(request.Email);
        if (user == null || !await _userService.ValidatePasswordAsync(user, request.Password))
        {
            throw new InvalidCredentialsException();
        }

        var accessToken = _tokenService.GenerateAccessToken(user);
        var refreshToken = await _tokenService.CreateRefreshTokenAsync(user);

        var expiryMinutes = int.Parse(_configuration["Jwt:ExpiryMinutes"] ?? "60");

        _logger.LogInformation("User logged in: {UserId}", user.Id);

        return Ok(new AuthResponse
        {
            User = MapUserResponse(user),
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token,
            ExpiresAt = DateTime.UtcNow.AddMinutes(expiryMinutes)
        });
    }

    [HttpPost("refresh")]
    [ProducesResponseType(typeof(AuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiError), StatusCodes.Status401Unauthorized)]
    public async Task<IActionResult> RefreshToken([FromBody] RefreshTokenRequest request)
    {
        var storedToken = await _tokenService.GetRefreshTokenAsync(request.RefreshToken);
        if (storedToken == null || !storedToken.IsActive)
        {
            throw new InvalidTokenException();
        }

        // Revoke old token and create new ones
        await _tokenService.RevokeRefreshTokenAsync(request.RefreshToken);

        var accessToken = _tokenService.GenerateAccessToken(storedToken.User);
        var newRefreshToken = await _tokenService.CreateRefreshTokenAsync(storedToken.User);

        var expiryMinutes = int.Parse(_configuration["Jwt:ExpiryMinutes"] ?? "60");

        return Ok(new AuthResponse
        {
            User = MapUserResponse(storedToken.User),
            AccessToken = accessToken,
            RefreshToken = newRefreshToken.Token,
            ExpiresAt = DateTime.UtcNow.AddMinutes(expiryMinutes)
        });
    }

    [HttpPost("logout")]
    [ProducesResponseType(StatusCodes.Status204NoContent)]
    public async Task<IActionResult> Logout([FromBody] RefreshTokenRequest request)
    {
        await _tokenService.RevokeRefreshTokenAsync(request.RefreshToken);
        return NoContent();
    }

    [HttpGet("profile")]
    [Microsoft.AspNetCore.Authorization.Authorize]
    [ProducesResponseType(typeof(UserResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ApiError), StatusCodes.Status401Unauthorized)]
    public async Task<IActionResult> GetProfile()
    {
        var userIdClaim = User.FindFirst("sub")?.Value;
        if (string.IsNullOrEmpty(userIdClaim) || !Guid.TryParse(userIdClaim, out var userId))
        {
            throw new InvalidTokenException();
        }

        var user = await _userService.GetByIdAsync(userId);
        if (user == null)
        {
            throw new UserNotFoundException(userIdClaim);
        }

        return Ok(MapUserResponse(user));
    }

    private static UserResponse MapUserResponse(Models.User user)
    {
        return new UserResponse
        {
            Id = user.Id.ToString(),
            Email = user.Email,
            Phone = user.Phone,
            FirstName = user.FirstName,
            LastName = user.LastName,
            KycLevel = user.KycLevel.ToString(),
            IsEmailVerified = user.IsEmailVerified,
            IsPhoneVerified = user.IsPhoneVerified,
            MfaEnabled = user.MfaEnabled,
            CreatedAt = user.CreatedAt
        };
    }
}
