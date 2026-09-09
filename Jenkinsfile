pipeline {
    agent any

environment {
    SERVICE_ID = 'go-capcha-api'
    SERVICE_NAME = 'Go Capcha API'

    DEPLOY_DIR = 'D:\\microservices\\go-capcha-api'
    EXE_NAME = 'go-capcha-api.exe'

    APP_PORT = '9731'
    PYTHON_HOME = 'C:\\Tools\\Python312'
    GO_VERSION = '1.27.1'
    GO_ROOT = 'D:\\tools\\go'
    GO_ZIP = 'D:\\tools\\go.zip'

    PATH = "D:\\tools\\go\\bin;${env.PATH}"
}

    stages {

        stage('Checkout') {
            steps {
                checkout scm
            }
        }

                stage('Check Environment') {
            steps {
                bat '''
                    SET PATH=%PYTHON_HOME%;%PYTHON_HOME%\\Scripts;%PATH%

                    echo ==============================
                    echo PYTHON
                    echo ==============================

                    python --version
                    python -m pip --version

                    echo ==============================
                    echo GIT
                    echo ==============================

                    git --version
                '''
            }
        }

stage('Install Go') {
    steps {
        powershell '''
            $ErrorActionPreference = "Stop"

            $goExe = "$env:GO_ROOT\\bin\\go.exe"

            if (Test-Path $goExe) {
                Write-Host "Go already installed:"
                & $goExe version
                exit 0
            }

            Write-Host "Go not found. Installing..."

            if (-not (Test-Path "D:\\tools")) {
                New-Item `
                    -ItemType Directory `
                    -Path "D:\\tools" `
                    -Force | Out-Null
            }

            [Net.ServicePointManager]::SecurityProtocol = `
                [Net.SecurityProtocolType]::Tls12

            $url = "https://go.dev/dl/go$env:GO_VERSION.windows-amd64.zip"

            Write-Host "Downloading:"
            Write-Host $url

            Invoke-WebRequest `
                -Uri $url `
                -OutFile "$env:GO_ZIP" `
                -UseBasicParsing

            if (-not (Test-Path "$env:GO_ZIP")) {
                throw "Go zip was not downloaded"
            }

            $tempDir = "D:\\tools\\go-temp"

            if (Test-Path $tempDir) {
                Remove-Item `
                    $tempDir `
                    -Recurse `
                    -Force
            }

            if (Test-Path "$env:GO_ROOT") {
                Remove-Item `
                    "$env:GO_ROOT" `
                    -Recurse `
                    -Force
            }

            Write-Host "Extracting Go..."

            Expand-Archive `
                -Path "$env:GO_ZIP" `
                -DestinationPath $tempDir `
                -Force

            Move-Item `
                "$tempDir\\go" `
                "$env:GO_ROOT"

            Remove-Item `
                $tempDir `
                -Recurse `
                -Force

            Remove-Item `
                "$env:GO_ZIP" `
                -Force

            Write-Host "Go installed successfully"

            & "$env:GO_ROOT\\bin\\go.exe" version
        '''
    }
}

        stage('Go Version') {
            steps {
                withEnv(["PATH+GO=${env.GO_ROOT}\\bin"]) {
                    bat '''
                        go version
                    '''
                }
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
                    "%PYTHON_HOME%\\python.exe" ^
                        "%SERVICE_MANAGER%" ^
                        stop ^
                        "%SERVICE_ID%"
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
                    "%PYTHON_HOME%\\python.exe"  "%SERVICE_MANAGER%" install ^
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
                    "%PYTHON_HOME%\\python.exe"  "%SERVICE_MANAGER%" start "%SERVICE_ID%"
                '''
            }
        }

        stage('Service Status') {
            steps {
                bat '''
                    "%PYTHON_HOME%\\python.exe" "%SERVICE_MANAGER%" status "%SERVICE_ID%"
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
                "%PYTHON_HOME%\\python.exe" "%SERVICE_MANAGER%" status "%SERVICE_ID%"
            '''
        }
    }
}