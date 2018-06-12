{% for target in targets %}
job('kubevirt-functional-tests-{{ target }}') {
    {% if target == "windows" %}
       label('windows')
    {% endif %}
    throttleConcurrentBuilds {
        maxTotal(0)
        maxPerNode(1)
    }
    concurrentBuild()
    wrappers {
        timeout {
          absolute(minutes = 180)
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
                              sourceFiles('{{ target }}-console.log')
                                }
                            }
                        }
                    }
                     hostConfigurationAccess(class: 'jenkins.plugins.publish_over_ssh.BapSshAlwaysRunPublisherPlugin', reference: '../..')
                    }
                }
            }
        }
        preBuildCleanup()
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
                cleanAfterCheckout()
            }
        }
    }
    triggers {
        githubPullRequest {
            admins(['booxter', 'cynepco3hahue', 'davidvossel', 'fabiand', 'rmohr', 'slintes', 'stu-gott', 'vladikr'])
            cron('H/2 * * * *')
            triggerPhrase('OK to test')
            allowMembersOfWhitelistedOrgsAsAdmin()
            extensions {
                commitStatus {
                    context('kubevirt-functional-tests/jenkins/pr/{{ target }}')
                    triggeredStatus('kubevirt-functional-tests/jenkins/pr {{ target }}')
                    startedStatus('kubevirt-functional-tests/jenkins/pr {{ target }}')
                    statusUrl('{{ storeReportUrl }}/$BUILD_NUMBER/{{ target }}-console.log')
                    completedStatus('SUCCESS', 'All is well')
                    completedStatus('FAILURE', 'Something went wrong. Investigate!')
                    completedStatus('PENDING', 'still in progress...')
                    completedStatus('ERROR', 'Something went really wrong. Investigate!')
                }
            }
        }
    }
    configure { node ->
        node / 'triggers' / 'org.jenkinsci.plugins.ghprb.GhprbTrigger' / 'extensions' << 'org.jenkinsci.plugins.ghprb.extensions.build.GhprbCancelBuildsOnUpdate' {
            overrideGlobal(false)
        }
    }
    steps {
        shell('''#!/bin/bash
set -o pipefail
export TARGET={{ target }}
cd go/src/kubevirt.io/kubevirt && bash automation/test.sh 2>&1 | tee ${WORKSPACE}/{{ target }}-console.log''')
    }
    publishers {
        xUnitPublisher {
            thresholdMode(1)
            testTimeMargin("")
            tools {
                jUnitType {
                    pattern('junit.xml')
                    skipNoTestFiles(false)
                    failIfNotNew(true)
                    deleteOutputFiles(true)
                    stopProcessingIfError(true)
                }
            }
            thresholds {
                failedThreshold {
                    unstableThreshold("")
                    unstableNewThreshold("")
                    failureThreshold("")
                    failureNewThreshold("")
                }
                skippedThreshold {
                    unstableThreshold("")
                    unstableNewThreshold("")
                    failureThreshold("")
                    failureNewThreshold("")
                }
            }
        }
    }
}
{% endfor %}
