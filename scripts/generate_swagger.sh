echo "Generating Swagger documentation..."
swag init -g cmd/main.go --exclude internal/model --parseDependency --parseInternal

if [ $? -eq 0 ]; then
    echo "Swagger documentation generated successfully!"
    echo "You can access the Swagger UI at: http://localhost:8080/swagger/index.html"
else
    echo "Error generating Swagger documentation"
    exit 1
fi