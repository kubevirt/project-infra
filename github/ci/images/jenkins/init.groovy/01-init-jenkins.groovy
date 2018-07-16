import jenkins.model.*
import hudson.util.*
import jenkins.install.*
import java.util.logging.Logger

// Set Jenkins URL
def jenkinsLocationConfiguration = JenkinsLocationConfiguration.get()
if (jenkinsLocationConfiguration.getUrl() != "{{ jenkinsUrl }}") {
    jenkinsLocationConfiguration.setUrl("{{ jenkinsUrl }}")
    jenkinsLocationConfiguration.save()
}

def instance = Jenkins.getInstance()

// Set Jenkins state to INITIAL_SETUP_COMPLETED
instance.setInstallState(InstallState.INITIAL_SETUP_COMPLETED)

// Install all needed plugins
def logger = Logger.getLogger("")
def installed = false
def initialized = false

def plugins = [
    "job-dsl",
    "github-oauth",
    "swarm",
    "throttle-concurrents",
    "publish-over-ssh",
    "build-timeout",
    "xunit",
    "test-results-analyzer",
    "ws-cleanup",
    "cloudbees-folder",
    "antisamy-markup-formatter",
    "credentials-binding",
    "timestamper",
    "workflow-aggregator",
    "github-branch-source",
    "pipeline-github-lib",
    "pipeline-stage-view",
    "git",
    "subversion",
    "matrix-auth",
    "pam-auth",
    "mailer",
]

logger.info("" + plugins)
def pm = instance.getPluginManager()
def uc = instance.getUpdateCenter()
uc.updateAllSites()

LinkedList futures = []

logger.info("Updating all plugins")
uc.getUpdates().each {
  logger.info("Updating " + it.getDisplayName())
  futures << it.deploy(true)
  installed = true
}

logger.info("Install missing plugins")
plugins.each {
  logger.info("Checking " + it)
  if (!pm.getPlugin(it)) {
    logger.info("Looking UpdateCenter for " + it)
    if (!initialized) {
      print it
      uc.updateAllSites()
      initialized = true
    }
    def plugin = uc.getPlugin(it)
    if (plugin) {
      logger.info("Installing " + it)
      futures << plugin.deploy(true)
      installed = true
    }
  }
}

def done = false

while(!done) {
  done = true
  logger.info("Checking if plugin is installed.")
  futures.each {
    if(it.isDone()) {
      logger.info("Plugin installed.")
    }
    done = done && it.isDone()
  }
  sleep(2000)
}

if (installed) {
  logger.info("All Plugins installed, initializing a restart!")
  instance.save()
  instance.doSafeRestart()
  sleep(20000)
}