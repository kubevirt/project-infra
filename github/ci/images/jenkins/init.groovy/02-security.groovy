import java.util.*

import hudson.model.User
import hudson.security.*

import jenkins.CLI;
import jenkins.model.*
import jenkins.security.s2m.AdminWhitelistRule

import com.cloudbees.plugins.credentials.impl.*
import com.cloudbees.plugins.credentials.*
import com.cloudbees.plugins.credentials.domains.*


def instance = Jenkins.getInstance()

// Give slaves less access on the master
Jenkins.instance.getInjector().getInstance(AdminWhitelistRule.class).setMasterKillSwitch(false)

// Disable CLI remoting
CLI.get().setEnabled(false);

// https://wiki.jenkins.io/display/JENKINS/CSRF+Protection
instance.setCrumbIssuer(new csrf.DefaultCrumbIssuer(true))

// Set security details
def hudsonRealm = new HudsonPrivateSecurityRealm(false)
hudsonRealm.createAccount("{{ jenkinsUser }}","{{ jenkinsPass }}")
instance.setSecurityRealm(hudsonRealm)

// Disable old Non-Encrypted protocols
HashSet<String> newProtocols = new HashSet<>(instance.getAgentProtocols());
newProtocols.removeAll(Arrays.asList(
        "JNLP3-connect", "JNLP2-connect", "JNLP-connect", "CLI-connect"
));
instance.setAgentProtocols(newProtocols);

def strategy = new FullControlOnceLoggedInAuthorizationStrategy()
instance.setAuthorizationStrategy(strategy)

// Don't run builds through master. We can build also on master through the swarm plugin
instance.setNumExecutors(0)

instance.save()
