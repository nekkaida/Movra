# Auth Service REST API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add REST API endpoints for external clients (mobile/web apps) to register, login, refresh tokens, and manage profiles.

**Architecture:** Controllers delegate to existing UserService and TokenService. DTOs handle request/response serialization with validation. Global exception handling returns consistent error responses. gRPC remains for internal service-to-service calls.

**Tech Stack:** .NET 8, ASP.NET Core, Entity Framework Core, PostgreSQL, BCrypt, JWT

---

## Phase A: DTOs and Exception Handling

### Task 1: Create Request/Response DTOs

**Files:**
- Create: `src/Dtos/RegisterRequest.cs`
- Create: `src/Dtos/LoginRequest.cs`
- Create: `src/Dtos/RefreshTokenRequest.cs`
- Create: `src/Dtos/AuthResponse.cs`
- Create: `src/Dtos/UserResponse.cs`
- Create: `src/Dtos/ApiError.cs`

**Step 1: Create RegisterRequest DTO**

```csharp
using System.ComponentModel.DataAnnotations;

namespace AuthService.Dtos;

public class RegisterRequest
{
    [Required(ErrorMessage = "Email is required")]
    [EmailAddress(ErrorMessage = "Invalid email format")]
    [MaxLength(255)]
    public string Email { get; set; } = string.Empty;

    [Required(ErrorMessage = "Password is required")]
    [MinLength(8, ErrorMessage = "Password must be at least 8 characters")]
    [MaxLength(100)]
    public string Password { get; set; } = string.Empty;

    [Required(ErrorMessage = "First name is required")]
    [MaxLength(100)]
    public string FirstName { get; set; } = string.Empty;

    [Required(ErrorMessage = "Last name is required")]
    [MaxLength(100)]
    public string LastName { get; set; } = string.Empty;

    [Phone(ErrorMessage = "Invalid phone format")]
    [MaxLength(20)]
    public string? Phone { get; set; }
}
```

**Step 2: Create LoginRequest DTO**

```csharp
using System.ComponentModel.DataAnnotations;

namespace AuthService.Dtos;

public class LoginRequest
{
    [Required(ErrorMessage = "Email is required")]
    [EmailAddress(ErrorMessage = "Invalid email format")]
    public string Email { get; set; } = string.Empty;

    [Required(ErrorMessage = "Password is required")]
    public string Password { get; set; } = string.Empty;
}
```

**Step 3: Create RefreshTokenRequest DTO**

```csharp
using System.ComponentModel.DataAnnotations;

namespace AuthService.Dtos;

public class RefreshTokenRequest
{
    [Required(ErrorMessage = "Refresh token is required")]
    public string RefreshToken { get; set; } = string.Empty;
}
```

**Step 4: Create AuthResponse DTO**

```csharp
namespace AuthService.Dtos;

public class AuthResponse
{
    public UserResponse User { get; set; } = null!;
    public string AccessToken { get; set; } = string.Empty;
    public string RefreshToken { get; set; } = string.Empty;
    public DateTime ExpiresAt { get; set; }
}
```

**Step 5: Create UserResponse DTO**

```csharp
namespace AuthService.Dtos;

public class UserResponse
{
    public string Id { get; set; } = string.Empty;
    public string Email { get; set; } = string.Empty;
    public string? Phone { get; set; }
    public string FirstName { get; set; } = string.Empty;
    public string LastName { get; set; } = string.Empty;
    public string KycLevel { get; set; } = string.Empty;
    public bool IsEmailVerified { get; set; }
    public bool IsPhoneVerified { get; set; }
    public bool MfaEnabled { get; set; }
    public DateTime CreatedAt { get; set; }
}
```

**Step 6: Create ApiError DTO**

```csharp
namespace AuthService.Dtos;

public class ApiError
{
    public string Code { get; set; } = string.Empty;
    public string Message { get; set; } = string.Empty;
    public Dictionary<string, string>? Details { get; set; }
    public DateTime Timestamp { get; set; } = DateTime.UtcNow;
    public string? Path { get; set; }
}
```

**Step 7: Commit**

```bash
git add src/Dtos/
git commit -m "feat(auth): add request/response DTOs with validation"
```

---

### Task 2: Create Global Exception Handler

**Files:**
- Create: `src/Exceptions/AuthException.cs`
- Create: `src/Middleware/ExceptionHandlerMiddleware.cs`
- Modify: `Program.cs`

**Step 1: Create custom exceptions**

```csharp
namespace AuthService.Exceptions;

public class AuthException : Exception
{
    public string Code { get; }
    public int StatusCode { get; }

    public AuthException(string code, string message, int statusCode = 400) : base(message)
    {
        Code = code;
        StatusCode = statusCode;
    }
}

public class InvalidCredentialsException : AuthException
{
    public InvalidCredentialsException()
        : base("INVALID_CREDENTIALS", "Invalid email or password", 401) { }
}

public class UserAlreadyExistsException : AuthException
{
    public UserAlreadyExistsException(string email)
        : base("USER_EXISTS", $"User with email {email} already exists", 409) { }
}

public class UserNotFoundException : AuthException
{
    public UserNotFoundException(string userId)
        : base("USER_NOT_FOUND", $"User not found: {userId}", 404) { }
}

public class InvalidTokenException : AuthException
{
    public InvalidTokenException()
        : base("INVALID_TOKEN", "Invalid or expired token", 401) { }
}
```

**Step 2: Create exception handler middleware**

```csharp
using System.Text.Json;
using AuthService.Dtos;
using AuthService.Exceptions;

namespace AuthService.Middleware;

public class ExceptionHandlerMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<ExceptionHandlerMiddleware> _logger;

    public ExceptionHandlerMiddleware(RequestDelegate next, ILogger<ExceptionHandlerMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await _next(context);
        }
        catch (AuthException ex)
        {
            _logger.LogWarning(ex, "Auth exception: {Code}", ex.Code);
            await HandleExceptionAsync(context, ex.StatusCode, ex.Code, ex.Message);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Unexpected error");
            await HandleExceptionAsync(context, 500, "INTERNAL_ERROR", "An unexpected error occurred");
        }
    }

    private static async Task HandleExceptionAsync(HttpContext context, int statusCode, string code, string message)
    {
        context.Response.ContentType = "application/json";
        context.Response.StatusCode = statusCode;

        var error = new ApiError
        {
            Code = code,
            Message = message,
            Timestamp = DateTime.UtcNow,
            Path = context.Request.Path
        };

        var options = new JsonSerializerOptions { PropertyNamingPolicy = JsonNamingPolicy.CamelCase };
        await context.Response.WriteAsync(JsonSerializer.Serialize(error, options));
    }
}

public static class ExceptionHandlerMiddlewareExtensions
{
    public static IApplicationBuilder UseGlobalExceptionHandler(this IApplicationBuilder app)
    {
        return app.UseMiddleware<ExceptionHandlerMiddleware>();
    }
}
```

**Step 3: Update Program.cs to use middleware**

Add after `var app = builder.Build();`:

```csharp
// Add this line after var app = builder.Build();
app.UseGlobalExceptionHandler();
```

**Step 4: Commit**

```bash
git add src/Exceptions/ src/Middleware/ Program.cs
git commit -m "feat(auth): add global exception handling middleware"
```

---

## Phase B: REST Controllers

### Task 3: Create AuthController

**Files:**
- Create: `src/Controllers/AuthController.cs`

**Step 1: Create AuthController**

```csharp
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
```

**Step 2: Commit**

```bash
git add src/Controllers/AuthController.cs
git commit -m "feat(auth): add REST AuthController with register, login, refresh, logout, profile"
```

---

### Task 4: Create HealthController

**Files:**
- Create: `src/Controllers/HealthController.cs`

**Step 1: Create HealthController**

```csharp
using Microsoft.AspNetCore.Mvc;

namespace AuthService.Controllers;

[ApiController]
public class HealthController : ControllerBase
{
    [HttpGet("/health")]
    public IActionResult Health()
    {
        return Ok(new { status = "healthy", service = "auth-service" });
    }

    [HttpGet("/ready")]
    public IActionResult Ready()
    {
        return Ok(new { status = "ready", service = "auth-service" });
    }
}
```

**Step 2: Commit**

```bash
git add src/Controllers/HealthController.cs
git commit -m "feat(auth): add health check endpoints"
```

---

## Phase C: Update UserService with Better Exceptions

### Task 5: Update UserService to throw custom exceptions

**Files:**
- Modify: `src/Services/UserService.cs`

**Step 1: Update UserService.CreateAsync**

Replace the exception throwing in CreateAsync:

```csharp
// Change this line in CreateAsync:
// throw new InvalidOperationException("User with this email already exists");
// To:
throw new UserAlreadyExistsException(email);
```

Also add the using statement at the top:
```csharp
using AuthService.Exceptions;
```

**Step 2: Commit**

```bash
git add src/Services/UserService.cs
git commit -m "refactor(auth): use custom exceptions in UserService"
```

---

## Phase D: Testing

### Task 6: Add Test Project and Unit Tests

**Files:**
- Create: `AuthService.Tests/AuthService.Tests.csproj`
- Create: `AuthService.Tests/Controllers/AuthControllerTests.cs`

**Step 1: Create test project file**

```xml
<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <IsPackable>false</IsPackable>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.NET.Test.Sdk" Version="17.8.0" />
    <PackageReference Include="xunit" Version="2.6.2" />
    <PackageReference Include="xunit.runner.visualstudio" Version="2.5.4" />
    <PackageReference Include="Moq" Version="4.20.70" />
    <PackageReference Include="FluentAssertions" Version="6.12.0" />
    <PackageReference Include="Microsoft.AspNetCore.Mvc.Testing" Version="8.0.0" />
  </ItemGroup>

  <ItemGroup>
    <ProjectReference Include="..\AuthService.csproj" />
  </ItemGroup>

</Project>
```

**Step 2: Create AuthControllerTests**

```csharp
using AuthService.Controllers;
using AuthService.Dtos;
using AuthService.Exceptions;
using AuthService.Models;
using AuthService.Services;
using FluentAssertions;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;
using Moq;
using Xunit;

namespace AuthService.Tests.Controllers;

public class AuthControllerTests
{
    private readonly Mock<IUserService> _userServiceMock;
    private readonly Mock<ITokenService> _tokenServiceMock;
    private readonly Mock<ILogger<AuthController>> _loggerMock;
    private readonly Mock<IConfiguration> _configMock;
    private readonly AuthController _controller;

    public AuthControllerTests()
    {
        _userServiceMock = new Mock<IUserService>();
        _tokenServiceMock = new Mock<ITokenService>();
        _loggerMock = new Mock<ILogger<AuthController>>();
        _configMock = new Mock<IConfiguration>();

        _configMock.Setup(c => c["Jwt:ExpiryMinutes"]).Returns("60");

        _controller = new AuthController(
            _userServiceMock.Object,
            _tokenServiceMock.Object,
            _loggerMock.Object,
            _configMock.Object
        );
    }

    [Fact]
    public async Task Register_NewUser_ReturnsCreated()
    {
        // Arrange
        var request = new RegisterRequest
        {
            Email = "test@example.com",
            Password = "password123",
            FirstName = "John",
            LastName = "Doe"
        };

        var user = new User
        {
            Id = Guid.NewGuid(),
            Email = request.Email,
            FirstName = request.FirstName,
            LastName = request.LastName
        };

        var refreshToken = new RefreshToken
        {
            Token = "refresh-token",
            ExpiresAt = DateTime.UtcNow.AddDays(7)
        };

        _userServiceMock.Setup(s => s.GetByEmailAsync(request.Email))
            .ReturnsAsync((User?)null);
        _userServiceMock.Setup(s => s.CreateAsync(
            request.Email, request.Password, request.FirstName, request.LastName, null))
            .ReturnsAsync(user);
        _tokenServiceMock.Setup(s => s.GenerateAccessToken(user))
            .Returns("access-token");
        _tokenServiceMock.Setup(s => s.CreateRefreshTokenAsync(user))
            .ReturnsAsync(refreshToken);

        // Act
        var result = await _controller.Register(request);

        // Assert
        result.Should().BeOfType<CreatedAtActionResult>();
        var createdResult = (CreatedAtActionResult)result;
        var response = createdResult.Value as AuthResponse;
        response.Should().NotBeNull();
        response!.AccessToken.Should().Be("access-token");
        response.RefreshToken.Should().Be("refresh-token");
    }

    [Fact]
    public async Task Register_ExistingUser_ThrowsException()
    {
        // Arrange
        var request = new RegisterRequest
        {
            Email = "existing@example.com",
            Password = "password123",
            FirstName = "John",
            LastName = "Doe"
        };

        var existingUser = new User { Id = Guid.NewGuid(), Email = request.Email };

        _userServiceMock.Setup(s => s.GetByEmailAsync(request.Email))
            .ReturnsAsync(existingUser);

        // Act & Assert
        await Assert.ThrowsAsync<UserAlreadyExistsException>(
            () => _controller.Register(request));
    }

    [Fact]
    public async Task Login_ValidCredentials_ReturnsOk()
    {
        // Arrange
        var request = new LoginRequest
        {
            Email = "test@example.com",
            Password = "password123"
        };

        var user = new User
        {
            Id = Guid.NewGuid(),
            Email = request.Email,
            FirstName = "John",
            LastName = "Doe"
        };

        var refreshToken = new RefreshToken
        {
            Token = "refresh-token",
            ExpiresAt = DateTime.UtcNow.AddDays(7)
        };

        _userServiceMock.Setup(s => s.GetByEmailAsync(request.Email))
            .ReturnsAsync(user);
        _userServiceMock.Setup(s => s.ValidatePasswordAsync(user, request.Password))
            .ReturnsAsync(true);
        _tokenServiceMock.Setup(s => s.GenerateAccessToken(user))
            .Returns("access-token");
        _tokenServiceMock.Setup(s => s.CreateRefreshTokenAsync(user))
            .ReturnsAsync(refreshToken);

        // Act
        var result = await _controller.Login(request);

        // Assert
        result.Should().BeOfType<OkObjectResult>();
        var okResult = (OkObjectResult)result;
        var response = okResult.Value as AuthResponse;
        response.Should().NotBeNull();
        response!.AccessToken.Should().Be("access-token");
    }

    [Fact]
    public async Task Login_InvalidCredentials_ThrowsException()
    {
        // Arrange
        var request = new LoginRequest
        {
            Email = "test@example.com",
            Password = "wrongpassword"
        };

        _userServiceMock.Setup(s => s.GetByEmailAsync(request.Email))
            .ReturnsAsync((User?)null);

        // Act & Assert
        await Assert.ThrowsAsync<InvalidCredentialsException>(
            () => _controller.Login(request));
    }

    [Fact]
    public async Task RefreshToken_ValidToken_ReturnsNewTokens()
    {
        // Arrange
        var request = new RefreshTokenRequest { RefreshToken = "valid-refresh-token" };

        var user = new User
        {
            Id = Guid.NewGuid(),
            Email = "test@example.com",
            FirstName = "John",
            LastName = "Doe"
        };

        var storedToken = new RefreshToken
        {
            Token = request.RefreshToken,
            User = user,
            ExpiresAt = DateTime.UtcNow.AddDays(7)
        };

        var newRefreshToken = new RefreshToken
        {
            Token = "new-refresh-token",
            ExpiresAt = DateTime.UtcNow.AddDays(7)
        };

        _tokenServiceMock.Setup(s => s.GetRefreshTokenAsync(request.RefreshToken))
            .ReturnsAsync(storedToken);
        _tokenServiceMock.Setup(s => s.GenerateAccessToken(user))
            .Returns("new-access-token");
        _tokenServiceMock.Setup(s => s.CreateRefreshTokenAsync(user))
            .ReturnsAsync(newRefreshToken);

        // Act
        var result = await _controller.RefreshToken(request);

        // Assert
        result.Should().BeOfType<OkObjectResult>();
        var okResult = (OkObjectResult)result;
        var response = okResult.Value as AuthResponse;
        response.Should().NotBeNull();
        response!.AccessToken.Should().Be("new-access-token");
        response.RefreshToken.Should().Be("new-refresh-token");

        _tokenServiceMock.Verify(s => s.RevokeRefreshTokenAsync(request.RefreshToken), Times.Once);
    }

    [Fact]
    public async Task RefreshToken_InvalidToken_ThrowsException()
    {
        // Arrange
        var request = new RefreshTokenRequest { RefreshToken = "invalid-token" };

        _tokenServiceMock.Setup(s => s.GetRefreshTokenAsync(request.RefreshToken))
            .ReturnsAsync((RefreshToken?)null);

        // Act & Assert
        await Assert.ThrowsAsync<InvalidTokenException>(
            () => _controller.RefreshToken(request));
    }
}
```

**Step 3: Run tests**

```bash
cd services/auth-service/AuthService.Tests
dotnet test
```

Expected: All tests pass

**Step 4: Commit**

```bash
git add AuthService.Tests/
git commit -m "test(auth): add unit tests for AuthController"
```

---

## Final Summary

After completing all tasks, the Auth Service will have:

1. **DTOs** - RegisterRequest, LoginRequest, RefreshTokenRequest, AuthResponse, UserResponse, ApiError
2. **Custom Exceptions** - AuthException, InvalidCredentialsException, UserAlreadyExistsException, UserNotFoundException, InvalidTokenException
3. **Exception Middleware** - Global error handling with consistent API responses
4. **REST Controllers** - AuthController (register, login, refresh, logout, profile), HealthController
5. **Unit Tests** - AuthControllerTests with 6 test cases

**REST Endpoints:**
| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/auth/register` | Create account | No |
| POST | `/api/auth/login` | Login | No |
| POST | `/api/auth/refresh` | Refresh tokens | No |
| POST | `/api/auth/logout` | Revoke refresh token | No |
| GET | `/api/auth/profile` | Get user profile | Yes |
| GET | `/health` | Health check | No |
| GET | `/ready` | Readiness check | No |
