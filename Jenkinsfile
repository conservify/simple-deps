timestamps {
    node {
        stage ('git') {
            checkout([$class: 'GitSCM', branches: [[name: '*/master']], userRemoteConfigs: [[url: 'https://github.com/Conservify/simple-deps.git']]])
        }

        stage ('build') {
            sh """
export PATH=/usr/local/go/bin:$PATH
export GOPATH=`pwd`/../go
go get gopkg.in/src-d/go-git.v4
make
 """
        }

        stage ('install') {
            sh """
cp build/simple-deps ~/workspace/bin
cp *.template ~/workspace/bin
 """
        }

        stage ('archive') {
            archiveArtifacts allowEmptyArchive: false, artifacts: 'build/*'
        }
    }
}
