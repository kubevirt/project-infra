import jenkins.model.*
import jenkins.install.InstallState

// Set Jenkins URL
def jenkinsLocationConfiguration = JenkinsLocationConfiguration.get()
if (jenkinsLocationConfiguration.getUrl() != "{{ jenkinsUrl }}") {
    jenkinsLocationConfiguration.setUrl("{{ jenkinsUrl }}")
    jenkinsLocationConfiguration.save()
}

// Disable initial wizard
if (!jenkins.installState.isSetupComplete()) {
    InstallState.INITIAL_SETUP_COMPLETED.initializeState()
}
