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
