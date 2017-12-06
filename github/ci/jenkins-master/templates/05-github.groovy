import java.lang.reflect.Field
import jenkins.model.*
import org.jenkinsci.plugins.ghprb.*
import hudson.util.Secret 

import org.jenkinsci.plugins.ghprb.Ghprb
import com.cloudbees.plugins.credentials.CredentialsStore
import com.cloudbees.plugins.credentials.SystemCredentialsProvider
import com.cloudbees.plugins.credentials.domains.Domain
import java.net.URI

def descriptor = Jenkins.instance.getDescriptorByType(org.jenkinsci.plugins.ghprb.GhprbTrigger.DescriptorImpl.class)
Field auth = descriptor.class.getDeclaredField("githubAuth")
auth.setAccessible(true)

def serverUri = new URI("https://api.github.com");

// Delete old credentials
def provider = new SystemCredentialsProvider.StoreImpl();
def domain = new Domain(serverUri.getHost(), "Auto generated credentials domain", null);
provider.removeDomain(domain);

// Create  new credentials
def githubCredentials = Ghprb.createCredentials(serverUri.toString(), "{{ githubToken }}")
githubAuth = new ArrayList<GhprbGitHubAuth>(1)
githubAuth.add(new GhprbGitHubAuth(serverUri.toString(), "{{ githubCallbackUrl }}", githubCredentials , "kubevirt-bot", "aebe0886-5ead-47cd-9a13-18490b7a2831" , new Secret("{{ githubSecret }}")))
auth.set(descriptor, githubAuth)

// Disable request for testing phrase
Field requestPhrase = descriptor.class.getDeclaredField("requestForTestingPhrase")
requestPhrase.setAccessible(true)
requestPhrase.set(descriptor, "")

descriptor.save()
