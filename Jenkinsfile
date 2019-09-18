def ws = "/data/jenkins/workspace/${JOB_NAME}-${BUILD_NUMBER}"

pipeline {
  agent {
    node {
      label 'fsi-build-tests'
      customWorkspace "${ws}/go/src/github.com/newrelic/nri-prometheus"
    }
  }
  options {
    buildDiscarder(logRotator(numToKeepStr: '15'))
    ansiColor('xterm')
  }

  environment {
    GOPATH = "${ws}/go"
    PATH = "${GOPATH}/bin:${PATH}"
  }

  stages {
    stage('Dependencies') {
      steps {
        sh 'make deps'
      }
    }
    stage('CI') {
      parallel {
        stage('Linting and Validation') {
          steps {
            sh 'make validate'
          }
        }
        stage('Unit Tests') {
          steps {
            sh 'make test'
          }
        }
      }
    }
  }
}
