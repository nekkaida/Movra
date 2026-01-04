using Microsoft.EntityFrameworkCore;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.IdentityModel.Tokens;
using System.Text;
using Serilog;
using Prometheus;
using AuthService.Data;
using AuthService.Services;
using AuthService.Grpc;
using AuthService.Middleware;

var builder = WebApplication.CreateBuilder(args);

// Configure Serilog
Log.Logger = new LoggerConfiguration()
    .ReadFrom.Configuration(builder.Configuration)
    .Enrich.FromLogContext()
    .WriteTo.Console()
    .CreateLogger();

builder.Host.UseSerilog();

// Add services
builder.Services.AddDbContext<AuthDbContext>(options =>
    options.UseNpgsql(builder.Configuration.GetConnectionString("DefaultConnection")));

builder.Services.AddScoped<IUserService, UserService>();
builder.Services.AddScoped<ITokenService, TokenService>();

// JWT Authentication
var jwtKey = builder.Configuration["Jwt:Key"] ?? "default-dev-key-change-in-production";
var jwtIssuer = builder.Configuration["Jwt:Issuer"] ?? "movra-auth";

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidateAudience = true,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true,
            ValidIssuer = jwtIssuer,
            ValidAudience = jwtIssuer,
            IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwtKey))
        };
    });

builder.Services.AddAuthorization();

// gRPC
builder.Services.AddGrpc();

// Controllers
builder.Services.AddControllers();

// Health checks
builder.Services.AddHealthChecks()
    .AddDbContextCheck<AuthDbContext>();

// OpenAPI
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

var app = builder.Build();

// Development setup
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

// Middleware
app.UseGlobalExceptionHandler();
app.UseSerilogRequestLogging();
app.UseRouting();

// Prometheus metrics
app.UseHttpMetrics();

app.UseAuthentication();
app.UseAuthorization();

// Map endpoints
app.MapControllers();
app.MapGrpcService<AuthGrpcService>();
app.MapHealthChecks("/health");
app.MapMetrics();

// Database migration on startup (development only)
if (app.Environment.IsDevelopment())
{
    using var scope = app.Services.CreateScope();
    var db = scope.ServiceProvider.GetRequiredService<AuthDbContext>();
    db.Database.EnsureCreated();
}

Log.Information("Auth Service starting on port {Port}", builder.Configuration["Urls"] ?? "5001");

app.Run();
