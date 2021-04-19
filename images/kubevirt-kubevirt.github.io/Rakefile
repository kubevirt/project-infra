#encoding: utf-8
namespace :links do
    require 'html-proofer'
    require 'optparse'

    # ENABLE LINK CHECKING ON FILES STAGED FOR GIT COMMIT
    # (QUICK LINK CHECK)
    if ENV['QUICK_LINK_CHECK_ENABLED'].to_s.downcase == "true"
      directories = %w(content)
      merge_base = `git merge-base origin/source HEAD`.chomp
      files_ignore = `git diff -z --name-only #{merge_base}`.split("\0")
      files_ignore = files_ignore.select do |filename|
        next true if directories.include?(File.dirname(filename))
        filename.end_with?('.html') ||
        filename.end_with?('.markdown') ||
        filename.end_with?('.md')
      end.map { |f|
        # Verify regex at https://regex101.com
        Regexp.new('^((?!' + File.basename(f, File.extname(f)) + ').)*$')
      }

      if files_ignore.length == 0
          abort "No html, markdown, md files staged for git commit"
      end

      note = ''

      # BLACK MAGIC TO HIJACK ARG AS A TASK
      task ARGV.last.to_sym do ; end

    # LINK CHECKING ON ALL FILES
    else
      files_ignore = []
      note = '(This can take a few mins to run) '
    end

    desc 'Generate HTML of Kubevirt.io'
    task :build do
        if ARGV.length > 0
          if ARGV.include? "quiet"
            quiet = '-q'

            # BLACK MAGIC TO HIJACK ARG AS A TASK
            task ARGV.last.to_sym do ; end
          else
            quiet = ''

            # BLACK MAGIC TO HIJACK ARG AS A TASK
            task ARGV.last.to_sym do ; end
          end
        end

        puts
        puts "Building..."
        sh 'bundle exec jekyll build' + ' ' + String(quiet)
    end

    desc 'Checks html files for broken external links'
    task :test_external, [:ARGV] do
        # Verify regex at https://regex101.com
        options = {
            :assume_extension   => true,
            :log_level          => :info,
            :external_only      => true,
            :internal_domains   => ["https://instructor.labs.sysdeseng.com", "https://www.youtube.com"],
            :url_ignore         => [ /http(s)?:\/\/(www.)?katacoda.com.*/ ],
            :url_swap           => {'https://kubevirt.io/' => '',},
            :http_status_ignore => [0, 400, 429, 999]
        }

        parser = OptionParser.new
        parser.banner = "Usage: rake -- [arguments]"
        # Added option -u which will remove the url_swap option to from the map
        parser.on("-u", "--us", "Remove url_swap from htmlProofer") do |url_swap|
            options.delete(:url_swap)
        end

        args = parser.order!(ARGV) {}
        parser.parse!(args)

        puts
        puts "Checks html files for broken external links " + note + "..."
        HTMLProofer.check_directory("./_site", options).run
    end

    desc 'Checks html files for broken internal links'
    task :test_internal do
        options = {
            :assume_extension   => true,
            :allow_hash_href    => true,
            :log_level          => :info,
            :disable_external   => true,
            :http_status_ignore => [0, 200, 400, 429, 999]
        }

        puts
        puts "Checks html files for broken internal links " + note + "..."
        HTMLProofer.check_directory("./_site", options).run
    end


    desc 'Checks html files for links to nonexistant userguide selectors'
    task :userguide_selectors => :build do
        # Verify regex's at https://regex101.com
        options = {
            :log_level          => :debug,
            :checks_to_ignore   => [ "ScriptCheck", "ImageCheck" ],
            :assume_extension   => true,
            :only_4xx           => true,
            :allow_hash_href    => true,
            :enforce_https      => true,
            :check_external_hash => true,
            :external_only      => true,
            :url_ignore         => [
                                   /http(s)?:\/\/(?!(kubevirt.io\/user-guide)).*/
                                   ],
        }

        puts
        puts "Discovering links to userguide with selectors " + note + " ..."

        # BLACK MAGIC BEGINS RIGHT HERE ...
        io = StringIO.new
        $stdout = io

        HTMLProofer.check_directory("./_site", options).run

        # UNCOMMENT TO enable full output of HTMLProofer
        STDOUT.puts $stdout.string

        $stdout.string.each_line do |f|
            if f.include? "#"
                if f.strip.match("Received a 200 for")
                    f["Received a "] = ''
                    f["for "] = ''
                    f["  in "] = ','
                    f.sub!(/^[0-9]+ /,'')
                    STDOUT.puts f
                end
            end
        end
    end
end

desc 'The default task will execute all tests in a row'
task :default => ['links:build', 'links:test_external', 'links:test_internal']
