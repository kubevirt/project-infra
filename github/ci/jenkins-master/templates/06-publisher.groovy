import java.lang.reflect.Field
import jenkins.model.*
import hudson.util.Secret 

import jenkins.plugins.publish_over_ssh.BapSshHostConfiguration
import com.cloudbees.plugins.credentials.CredentialsStore
import com.cloudbees.plugins.credentials.SystemCredentialsProvider
import com.cloudbees.plugins.credentials.domains.Domain
import java.net.URI

def descriptor = Jenkins.instance.getDescriptorByType(jenkins.plugins.publish_over_ssh.descriptor.BapSshPublisherDescriptor.class)
def plugin = descriptor.getPublisherPluginDescriptor()

plugin.getCommonConfig().setKey("""{{ lookup('file', 'id_rsa_jenkins_master') }}""")

def config = new BapSshHostConfiguration()
config.setName("store")
config.setHostname("{{ storeSshUrl }}")
config.setUsername("{{ storeSshUser  }}")
config.setRemoteRootDir("{{ storeSshRemoteDir }}")
config.setPort(22)

plugin.removeHostConfiguration("store")
plugin.addHostConfiguration(config)
plugin.save()
