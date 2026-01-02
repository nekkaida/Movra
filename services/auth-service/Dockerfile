# Build stage
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy csproj and restore
COPY AuthService.csproj .
COPY ../../proto ./proto
RUN dotnet restore

# Copy source and build
COPY . .
RUN dotnet publish -c Release -o /app/publish

# Runtime stage
FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app

# Create non-root user
RUN adduser --disabled-password --gecos "" --uid 1001 appuser

COPY --from=build /app/publish .

USER appuser

EXPOSE 5001 5002

ENTRYPOINT ["dotnet", "AuthService.dll"]
