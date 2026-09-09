pipeline {
    agent any

    environment {
        SERVICE_ID = 'go-capcha-api'
        SERVICE_NAME = 'Go Capcha API'

        DEPLOY_DIR = 'D:\\microservices\\go-capcha-api'
        EXE_NAME = 'go-capcha-api.exe'

        SERVICE_MANAGER = 'D:\\tools\\service_manager.py'

        APP_PORT = '9731'
    }

    stages {

        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Go Version') {
            steps {
                bat '''
                    go version
                '''
            }
        }

        stage('Dependencies') {
            steps {
                bat '''
                    go mod download
                '''
            }
        }

        stage('Build') {
            steps {
                bat '''
                    echo ==========================================
                    echo Building Go application
                    echo ==========================================

                    if exist "%EXE_NAME%" (
                        del /F /Q "%EXE_NAME%"
                    )

                    go build -o "%EXE_NAME%" .

                    if not exist "%EXE_NAME%" (
                        echo ERROR: Executable was not generated
                        exit /B 1
                    )
                '''
            }
        }

        stage('Stop Service') {
            steps {
                bat '''
                    python "%SERVICE_MANAGER%" stop "%SERVICE_ID%"
                '''
            }
        }

        stage('Prepare Destination') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    if (-not (Test-Path "$env:DEPLOY_DIR")) {
                        New-Item `
                            -ItemType Directory `
                            -Path "$env:DEPLOY_DIR" `
                            -Force | Out-Null
                    }
                '''
            }
        }

        stage('Deploy') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    $source = Join-Path `
                        "$env:WORKSPACE" `
                        "$env:EXE_NAME"

                    $target = Join-Path `
                        "$env:DEPLOY_DIR" `
                        "$env:EXE_NAME"

                    Write-Host "Copying:"
                    Write-Host "  $source"
                    Write-Host "to:"
                    Write-Host "  $target"

                    Copy-Item `
                        $source `
                        $target `
                        -Force
                '''
            }
        }

        stage('Install / Configure Service') {
            steps {
                bat '''
                    python "%SERVICE_MANAGER%" install ^
                        "%SERVICE_ID%" ^
                        "%DEPLOY_DIR%" ^
                        --type go ^
                        --port %APP_PORT% ^
                        --executable "%EXE_NAME%" ^
                        --name "%SERVICE_NAME%" ^
                        --description "Go CAPTCHA API service"
                '''
            }
        }

        stage('Start Service') {
            steps {
                bat '''
                    python "%SERVICE_MANAGER%" start "%SERVICE_ID%"
                '''
            }
        }

        stage('Service Status') {
            steps {
                bat '''
                    python "%SERVICE_MANAGER%" status "%SERVICE_ID%"
                '''
            }
        }

        stage('Health Check') {
            steps {
                powershell '''
                    $ErrorActionPreference = "Stop"

                    $url = "http://localhost:$env:APP_PORT/health"

                    Write-Host "Checking $url"

                    $ok = $false

                    for ($i = 1; $i -le 10; $i++) {

                        try {

                            $response = Invoke-WebRequest `
                                -Uri $url `
                                -UseBasicParsing `
                                -TimeoutSec 5

                            if ($response.StatusCode -eq 200) {
                                Write-Host "Service is healthy"
                                Write-Host $response.Content

                                $ok = $true
                                break
                            }
                        }
                        catch {
                            Write-Host "Health check attempt $i failed"
                        }

                        Start-Sleep -Seconds 2
                    }

                    if (-not $ok) {
                        throw "go-capcha-api did not become healthy"
                    }
                '''
            }
        }
    }

    post {

        success {
            echo 'go-capcha-api deployed successfully'
        }

        failure {
            echo 'Deployment failed'

            bat '''
                python "%SERVICE_MANAGER%" status "%SERVICE_ID%"
            '''
        }
    }
}