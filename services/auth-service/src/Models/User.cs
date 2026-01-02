using System.ComponentModel.DataAnnotations;

namespace AuthService.Models;

public class User
{
    [Key]
    public Guid Id { get; set; } = Guid.NewGuid();

    [Required]
    [EmailAddress]
    [MaxLength(255)]
    public string Email { get; set; } = string.Empty;

    [Required]
    public string PasswordHash { get; set; } = string.Empty;

    [Phone]
    [MaxLength(20)]
    public string? Phone { get; set; }

    [Required]
    [MaxLength(100)]
    public string FirstName { get; set; } = string.Empty;

    [Required]
    [MaxLength(100)]
    public string LastName { get; set; } = string.Empty;

    public KycLevel KycLevel { get; set; } = KycLevel.None;

    public bool IsEmailVerified { get; set; } = false;

    public bool IsPhoneVerified { get; set; } = false;

    public bool MfaEnabled { get; set; } = false;

    public string? MfaSecret { get; set; }

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;

    public DateTime UpdatedAt { get; set; } = DateTime.UtcNow;

    public DateTime? LastLoginAt { get; set; }

    // Navigation properties
    public ICollection<RefreshToken> RefreshTokens { get; set; } = new List<RefreshToken>();
    public ICollection<KycRecord> KycRecords { get; set; } = new List<KycRecord>();
}

public enum KycLevel
{
    None = 0,
    Basic = 1,      // Email verified
    Verified = 2,   // ID verified
    Premium = 3     // Full verification + proof of address
}

public class RefreshToken
{
    [Key]
    public Guid Id { get; set; } = Guid.NewGuid();

    [Required]
    public Guid UserId { get; set; }

    [Required]
    public string Token { get; set; } = string.Empty;

    public DateTime ExpiresAt { get; set; }

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;

    public DateTime? RevokedAt { get; set; }

    public bool IsExpired => DateTime.UtcNow >= ExpiresAt;

    public bool IsRevoked => RevokedAt != null;

    public bool IsActive => !IsRevoked && !IsExpired;

    // Navigation
    public User User { get; set; } = null!;
}

public class KycRecord
{
    [Key]
    public Guid Id { get; set; } = Guid.NewGuid();

    [Required]
    public Guid UserId { get; set; }

    public KycLevel Level { get; set; }

    public KycStatus Status { get; set; } = KycStatus.Pending;

    public string? DocumentType { get; set; }

    public string? DocumentNumber { get; set; }

    public string? RejectionReason { get; set; }

    public DateTime SubmittedAt { get; set; } = DateTime.UtcNow;

    public DateTime? ReviewedAt { get; set; }

    // Navigation
    public User User { get; set; } = null!;
}

public enum KycStatus
{
    Pending = 0,
    Approved = 1,
    Rejected = 2
}
