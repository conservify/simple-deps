timestamps {
    node {
        stage ('simple-deps - Checkout') {
            checkout([$class: 'GitSCM', branches: [[name: '*/master']], userRemoteConfigs: [[url: 'https://github.com/Conservify/simple-deps.git']]]) 
        }
        stage ('simple-deps - Build') {
            sh """ 
export PATH=/usr/local/go/bin:$PATH
export GOPATH=`pwd`/../go
go get gopkg.in/src-d/go-git.v4
make
cp build/simple-deps ~/workspace/bin
cp *.template ~/workspace/bin 
 """
            archiveArtifacts allowEmptyArchive: false, artifacts: 'build/simple-deps'
        }
    }
}
