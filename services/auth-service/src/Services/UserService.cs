using Microsoft.EntityFrameworkCore;
using AuthService.Data;
using AuthService.Models;
using AuthService.Exceptions;
using BCrypt.Net;

namespace AuthService.Services;

public interface IUserService
{
    Task<User?> GetByIdAsync(Guid id);
    Task<User?> GetByEmailAsync(string email);
    Task<User> CreateAsync(string email, string password, string firstName, string lastName, string? phone);
    Task<bool> ValidatePasswordAsync(User user, string password);
    Task<User> UpdateKycLevelAsync(Guid userId, KycLevel level);
    Task<TransferLimits> GetTransferLimitsAsync(KycLevel level);
}

public class UserService : IUserService
{
    private readonly AuthDbContext _context;
    private readonly ILogger<UserService> _logger;

    public UserService(AuthDbContext context, ILogger<UserService> logger)
    {
        _context = context;
        _logger = logger;
    }

    public async Task<User?> GetByIdAsync(Guid id)
    {
        return await _context.Users.FindAsync(id);
    }

    public async Task<User?> GetByEmailAsync(string email)
    {
        return await _context.Users.FirstOrDefaultAsync(u => u.Email == email.ToLowerInvariant());
    }

    public async Task<User> CreateAsync(string email, string password, string firstName, string lastName, string? phone)
    {
        var existingUser = await GetByEmailAsync(email);
        if (existingUser != null)
        {
            throw new UserAlreadyExistsException(email);
        }

        var user = new User
        {
            Email = email.ToLowerInvariant(),
            PasswordHash = BCrypt.Net.BCrypt.HashPassword(password),
            FirstName = firstName,
            LastName = lastName,
            Phone = phone,
            KycLevel = KycLevel.None
        };

        _context.Users.Add(user);
        await _context.SaveChangesAsync();

        _logger.LogInformation("Created new user {UserId} with email {Email}", user.Id, user.Email);

        return user;
    }

    public Task<bool> ValidatePasswordAsync(User user, string password)
    {
        return Task.FromResult(BCrypt.Net.BCrypt.Verify(password, user.PasswordHash));
    }

    public async Task<User> UpdateKycLevelAsync(Guid userId, KycLevel level)
    {
        var user = await GetByIdAsync(userId);
        if (user == null)
        {
            throw new UserNotFoundException(userId.ToString());
        }

        user.KycLevel = level;
        user.UpdatedAt = DateTime.UtcNow;

        await _context.SaveChangesAsync();

        _logger.LogInformation("Updated KYC level for user {UserId} to {KycLevel}", userId, level);

        return user;
    }

    public Task<TransferLimits> GetTransferLimitsAsync(KycLevel level)
    {
        // Define limits based on KYC level
        var limits = level switch
        {
            KycLevel.None => new TransferLimits
            {
                DailyLimit = 0,
                MonthlyLimit = 0,
                PerTransactionLimit = 0,
                Currency = "SGD"
            },
            KycLevel.Basic => new TransferLimits
            {
                DailyLimit = 500,
                MonthlyLimit = 2000,
                PerTransactionLimit = 500,
                Currency = "SGD"
            },
            KycLevel.Verified => new TransferLimits
            {
                DailyLimit = 5000,
                MonthlyLimit = 20000,
                PerTransactionLimit = 5000,
                Currency = "SGD"
            },
            KycLevel.Premium => new TransferLimits
            {
                DailyLimit = 50000,
                MonthlyLimit = 200000,
                PerTransactionLimit = 50000,
                Currency = "SGD"
            },
            _ => new TransferLimits
            {
                DailyLimit = 0,
                MonthlyLimit = 0,
                PerTransactionLimit = 0,
                Currency = "SGD"
            }
        };

        return Task.FromResult(limits);
    }
}

public class TransferLimits
{
    public decimal DailyLimit { get; set; }
    public decimal MonthlyLimit { get; set; }
    public decimal PerTransactionLimit { get; set; }
    public string Currency { get; set; } = "SGD";
}
