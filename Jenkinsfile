timestamps {
    node {
        stage ('git') {
            checkout scm
        }

        stage ('build') {
            withEnv(["PATH+GOLANG=${tool 'golang-amd64'}/bin"]) {
                sh "make clean all"
            }
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
