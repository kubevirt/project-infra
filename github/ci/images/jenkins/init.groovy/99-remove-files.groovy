// This will delete sensetive files

def sensetive_files = [
    "/var/jenkins_home/init.groovy.d/01-init-jenkins.groovy",
    "/var/jenkins_home/init.groovy.d/02-security.groovy",
    "/var/jenkins_home/init.groovy.d/03-credentials.groovy"
]

for (filename in sensetive_files) {
    boolean fileSuccessfullyDeleted =  new File(filename).delete()    
}
