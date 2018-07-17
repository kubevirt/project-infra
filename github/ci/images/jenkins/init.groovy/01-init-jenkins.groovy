import jenkins.model.*
import jenkins.install.*

import hudson.util.*

// Set Jenkins URL
def jenkinsLocationConfiguration = JenkinsLocationConfiguration.get()
if (jenkinsLocationConfiguration.getUrl() != "{{ jenkinsUrl }}") {
    jenkinsLocationConfiguration.setUrl("{{ jenkinsUrl }}")
    jenkinsLocationConfiguration.save()
}

// Run safe restart to disable wizard
def instance = Jenkins.getInstance()
instance.doSafeRestart()
