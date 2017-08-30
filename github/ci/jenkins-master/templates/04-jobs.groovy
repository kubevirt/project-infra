job('kubevirt-functional-tests') {
    throttleConcurrentBuilds {
        maxTotal(0)
        maxPerNode(1)
    }
    concurrentBuild()
    wrappers {
        timeout {
          absolute(minutes = 120)
        }
        configure { node ->
            node / 'buildWrappers'  << 'jenkins.plugins.publish__over__ssh.BapSshPostBuildWrapper' {
                postBuild {
                    consolePrefix('SSH:')
                    delegate {
                    delegate.publishers {
                        'jenkins.plugins.publish__over__ssh.BapSshPublisher' {
                            configName('store')
                            transfers {
                                'jenkins.plugins.publish__over__ssh.BapSshTransfer' {
                             remoteDirectory('$BUILD_NUMBER')
                              sourceFiles('console.log')
                                }
                            }
                        }
                    }
                     hostConfigurationAccess(class: 'jenkins.plugins.publish_over_ssh.BapSshAlwaysRunPublisherPlugin', reference: '../..')
                    }
                }
            }
        }
    }
    parameters {
        stringParam('sha1', '', 'commit to build')
    }
    scm {
        git {
            remote {
                github('{{ githubRepo }}')
                refspec('+refs/pull/*:refs/remotes/origin/pr/*')
            }
            branch('${sha1}')
            extensions {
                relativeTargetDirectory('go/src/kubevirt.io/kubevirt')
            }
        }
    }
    triggers {
        githubPullRequest {
            admins(['rmohr', 'fabiand', 'stu-gott', 'admiyo', 'davidvossel', 'vladikr', 'berrange'])
            cron('H/2 * * * *')
            triggerPhrase('OK to test')
            allowMembersOfWhitelistedOrgsAsAdmin()
            extensions {
                commitStatus {
                    context('kubevirt-functional-tests/jenkins/pr')
                    triggeredStatus('kubevirt-functional-tests/jenkins/pr')
                    startedStatus('kubevirt-functional-tests/jenkins/pr')
                    statusUrl('{{ storeReportUrl }}/$BUILD_NUMBER/console.log')
                    completedStatus('SUCCESS', 'All is well')
                    completedStatus('FAILURE', 'Something went wrong. Investigate!')
                    completedStatus('PENDING', 'still in progress...')
                    completedStatus('ERROR', 'Something went really wrong. Investigate!')
                }
            }
        }
    }
    steps {
        shell('''#!/bin/bash
set -o pipefail
cd go/src/kubevirt.io/kubevirt && bash automation/test.sh 2>&1 | tee ${WORKSPACE}/console.log''')
    }
}
