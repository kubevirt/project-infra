import jenkins.model.*
import hudson.security.*
import jenkins.security.s2m.AdminWhitelistRule
import jenkins.CLI;


// Give slaves less access on the master
Jenkins.instance.getInjector().getInstance(AdminWhitelistRule.class)
.setMasterKillSwitch(false)

// Disable CLI remoting
CLI.get().setEnabled(false);


// Set security details
def instance = Jenkins.getInstance()

def hudsonRealm = new HudsonPrivateSecurityRealm(false)
// XXX, that should be done via jenkins and obviously without hardcoding
hudsonRealm.createAccount("{{ jenkinsUser }}","{{ jenkinsPass }}")
instance.setSecurityRealm(hudsonRealm)

def strategy = new FullControlOnceLoggedInAuthorizationStrategy()
instance.setAuthorizationStrategy(strategy)
instance.save()
