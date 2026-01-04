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
