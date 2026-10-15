pipeline {
    agent {
        docker { image 'golang:1.26.9' }
    }
    stages {
        stage('Test') {
            steps {
                sh 'go version'
                sh 'go vet ./...'
                sh 'go build ./...'
            }
        }
    }
}
