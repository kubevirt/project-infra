import jenkins.model.*

import com.cloudbees.plugins.credentials.impl.*
import com.cloudbees.plugins.credentials.*
import com.cloudbees.plugins.credentials.domains.*

import hudson.model.User
import hudson.security.ACL
import hudson.security.ACLContext

import org.acegisecurity.context.SecurityContext;
import org.acegisecurity.context.SecurityContextHolder;

// Create GitHub credentials
def systemCredentialsProvider = SystemCredentialsProvider.getInstance()
def credExists = false
for (cred in systemCredentialsProvider.getCredentials()) {
    if (cred.id == "github-token") {
        credExists = true
        break
    }
}

if (!credExists) {
    Credentials c = new UsernamePasswordCredentialsImpl(
        CredentialsScope.GLOBAL,
        "github-token",
        "GitHub access token",
        "{{ githubUser }}",
        "{{ githubToken }}"
    )
    systemCredentialsProvider.getStore().addCredentials(Domain.global(), c)
}

// Create blue-ocean domain and credentials
def userCredentialsProvider = new UserCredentialsProvider()
def user = User.getById("{{ jenkinsUser }}", false)
def credStore = userCredentialsProvider.getStore(user)

// Changes the Authentication associated with the current thread to the user one
SecurityContext old = ACL.impersonate(user.impersonate());

def domainExists = false
for (domain in credStore.getDomains()) {
    if (domain.name == "blueocean-github-domain") {
        domainExists = true
        break
    }
}

if (!domainExists) {
    def blueOceanDomain = new Domain(
        "blueocean-github-domain",
        "blueocean-github-domain to store credentials by BlueOcean",
        null
    )
    def blueOceanCred = new UsernamePasswordCredentialsImpl(
        CredentialsScope.GLOBAL,
        "github",
        "GitHub access token",
        "{{ jenkinsUser }}",
        "{{ githubToken }}"
    )
    credStore.addDomain(blueOceanDomain, [blueOceanCred])
    credStore.save()
}

// Restores old security context
SecurityContextHolder.setContext(old);
