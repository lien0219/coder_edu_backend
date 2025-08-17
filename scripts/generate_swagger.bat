@echo off
cd /d %~dp0..

echo Generating Swagger documentation...
swag init -g cmd/main.go --exclude internal/model --parseDependency --parseInternal

if %errorlevel% equ 0 (
    echo Swagger documentation generated successfully!
    echo You can access the Swagger UI at: http://localhost:8080/swagger/index.html
) else (
    echo Error generating Swagger documentation
)