import jenkins.model.*
import java.util.logging.Logger

def logger = Logger.getLogger("")
def installed = false
def initialized = false

def pluginParameter="ghprb job-dsl github-oauth swarm"
def plugins = pluginParameter.split()
logger.info("" + plugins)
def instance = Jenkins.getInstance()
def pm = instance.getPluginManager()
def uc = instance.getUpdateCenter()
uc.updateAllSites()

LinkedList futures = []

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
