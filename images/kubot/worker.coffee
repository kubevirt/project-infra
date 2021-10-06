YAML = require 'yaml'
fs = require 'fs'

module.exports = (robot) ->
  MAINTAINERS_KEY = "maintainers"

  fs.exists './kubeconfig', (exists) ->
    unless exists
      fs.cp '/etc/kubeconfig/config', './kubeconfig', { dereference: true }, (error) ->
        if error
          console.log "Could not copy kubeconfig: #{error}"

  check_kubectl_input = (subaction, args_str) ->
    allowed_subactions = ["get", "log", "logs", "describe", "config", "top"]
    if subaction not in allowed_subactions
      return false
    args = args_str.split(/\s+/)
    if subaction is "config" and args[0] is not "use-context"
        return false
    if /^logs?/.test subaction and "-f" in args
        return false
    if subaction is "get" and "-w" in args
        return false
    return true

  execute_action = (res, action, subaction, args) ->
    user = res.envelope.user.name
    maintainers = robot.brain.get(MAINTAINERS_KEY) || []
    if user not in maintainers
      console.log "`#{user}` is not allowed to perform `#{action}`"
      return
    if action is 'ctl'
      execute_kubectl_action res, subaction, args

  execute_kubectl_action = (res, action, args) ->
    if not check_kubectl_input action, args
      res.send "Could not execute `#{action} #{args}`"
      return
    res.send "Executing kubectl #{action} command..."
    exec = require('child_process').exec
    command = "kubectl #{action} #{args} --kubeconfig=./kubeconfig"
    exec command, (error, stdout, stderror) ->
      if stdout
        res.send "```\n" + stdout + "\n```"
      else
        res.send "Sorry that didn't work"
        if error
          res.send (error.stack)
        if stderror
          res.send (stderror)

  robot.respond /(\w+)\s*(\w+)\s*(.*)/i, (res) ->
    requested_action = res.match[1]
    requested_subaction = res.match[2]
    requested_args = res.match[3]

    maintainers = robot.brain.get(MAINTAINERS_KEY) || []
    if maintainers.length is 0
      console.log "Fetching maintainers..."
      maintainers_url = "https://raw.githubusercontent.com/kubevirt/project-infra/main/OWNERS_ALIASES"
      robot.http(maintainers_url)
        .get() (err, getRes, body) ->
          if err
            console.log("Could not read ci-maintainers from #{maintainers_url}: #{err}")
            return
          data = YAML.parse body
          maintainers = data.aliases['ci-maintainers']
          console.log "Fetched maintainers: #{maintainers}"
          robot.brain.set(MAINTAINERS_KEY, maintainers)
          execute_action res, requested_action, requested_subaction, requested_args
    else
      execute_action res, requested_action, requested_subaction, requested_args

  robot.router.get '/health', (req, res) ->
    res.setHeader 'content-type', 'text/plain'
    res.end 'OK'
