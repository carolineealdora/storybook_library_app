# code base

###### version : v1.0

### Programming Language
- [Go](https://go.dev/)

### Framework
- [Echo Go Framework](https://echo.labstack.com/)

### How to Deploy on Localhost
```
git pull origin development
go run main.go
```

### Environment Requirement
| Name     | Status | Description              | Version |
| -------  | ------ | ------------------------ | ------- |
| Database | New    | Database                 | 10gb    |

### Environment Variables
| Name | Value       | Description | Location |
|------|-------------|-------------|----------|
| ENV  | development | Server host | env      |

### PORT
| Port | Description      |
| ---- | ---------------- |
| 9999 | API service      |

### Build and run production
```
go get -d -v ./... 
go build -o /go/bin/app 
```

## Run Development
```
go run main.go
```

## Run Unit Test