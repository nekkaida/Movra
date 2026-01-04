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
