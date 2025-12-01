require "http/client"
require "file_utils"
require "option_parser"

module BytesCLI
  LIBS_DIR = File.expand_path("~/.hackeros/hacker-lang/libs")
  PLUGINS_DIR = File.expand_path("~/.hackeros/hacker-lang/plugins")
  SOURCES_DIR = File.expand_path("~/.hackeros/hacker-lang/source")
  TMP_DIR = "/tmp"
  REPO_LIBS_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
  REPO_PLUGINS_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/plugins-repo.hacker"
  REPO_SOURCES_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/source-repo.hacker"

  # Colors
  GREEN = "\e[32m"
  RED = "\e[31m"
  YELLOW = "\e[33m"
  RESET = "\e[0m"

  def self.ensure_dirs
    Dir.mkdir_p(LIBS_DIR)
    Dir.mkdir_p(PLUGINS_DIR)
    Dir.mkdir_p(SOURCES_DIR)
  end

  def self.fetch_repo_content(url : String) : String
    response = HTTP::Client.get(url)
    if response.success?
      response.body
    else
      raise "Failed to fetch repo: #{response.status_code}"
    end
  end

  def self.parse_repo(content : String) : Hash(String, String)
    repo = Hash(String, String).new
    current_section = ""
    content.lines.each do |line|
      line = line.strip
      next if line.empty?
      if line.ends_with?(":")
        current_section = line.chomp(":")
      elsif line.includes?(":")
        parts = line.split(":", 2)
        name = parts[0].strip
        url = parts[1].strip.gsub("\"", "")
        repo[name] = url
      end
    end
    repo
  end

  def self.install_lib(name : String)
    content = fetch_repo_content(REPO_LIBS_URL)
    libs = parse_repo(content)
    if url = libs[name]?
      puts "#{YELLOW}Downloading #{name} from #{url}...#{RESET}"
      response = HTTP::Client.get(url)
      if response.success?
        tmp_path = File.join(TMP_DIR, name)
        File.write(tmp_path, response.body)
        dest_path = File.join(LIBS_DIR, name)
        FileUtils.mv(tmp_path, dest_path)
        puts "#{GREEN}Installed #{name} to #{dest_path}#{RESET}"
      else
        puts "#{RED}Failed to download #{name}: #{response.status_code}#{RESET}"
      end
    else
      puts "#{RED}Library #{name} not found in repository#{RESET}"
    end
  end

  def self.remove_lib(name : String)
    path = File.join(LIBS_DIR, name)
    if File.exists?(path)
      File.delete(path)
      puts "#{GREEN}Removed #{name} from #{LIBS_DIR}#{RESET}"
    else
      puts "#{RED}Library #{name} not found#{RESET}"
    end
  end

  def self.install_plugin(name : String)
    content = fetch_repo_content(REPO_PLUGINS_URL)
    plugins = parse_repo(content)
    if url = plugins[name]?
      puts "#{YELLOW}Cloning #{name} from #{url}...#{RESET}"
      dest_dir = File.join(PLUGINS_DIR, name)
      if Dir.exists?(dest_dir)
        puts "#{YELLOW}Plugin #{name} already exists, pulling updates...#{RESET}"
        Process.run("git", ["-C", dest_dir, "pull"], output: STDOUT, error: STDERR)
      else
        Process.run("git", ["clone", "#{url}.git", dest_dir], output: STDOUT, error: STDERR)
      end
      puts "#{GREEN}Installed plugin #{name} to #{dest_dir}#{RESET}"
    else
      puts "#{RED}Plugin #{name} not found in repository#{RESET}"
    end
  end

  def self.remove_plugin(name : String)
    path = File.join(PLUGINS_DIR, name)
    if Dir.exists?(path)
      FileUtils.rm_rf(path)
      puts "#{GREEN}Removed plugin #{name} from #{PLUGINS_DIR}#{RESET}"
    else
      puts "#{RED}Plugin #{name} not found#{RESET}"
    end
  end

  def self.install_source(name : String)
    content = fetch_repo_content(REPO_SOURCES_URL)
    sources = parse_repo(content)
    if url = sources[name]?
      puts "#{YELLOW}Cloning #{name} from #{url}...#{RESET}"
      dest_dir = File.join(SOURCES_DIR, name)
      if Dir.exists?(dest_dir)
        puts "#{YELLOW}Source #{name} already exists, pulling updates...#{RESET}"
        Process.run("git", ["-C", dest_dir, "pull"], output: STDOUT, error: STDERR)
      else
        Process.run("git", ["clone", url, dest_dir], output: STDOUT, error: STDERR)
      end
      puts "#{GREEN}Installed source #{name} to #{dest_dir}#{RESET}"
    else
      puts "#{RED}Source #{name} not found in repository#{RESET}"
    end
  end

  def self.remove_source(name : String)
    path = File.join(SOURCES_DIR, name)
    if Dir.exists?(path)
      FileUtils.rm_rf(path)
      puts "#{GREEN}Removed source #{name} from #{SOURCES_DIR}#{RESET}"
    else
      puts "#{RED}Source #{name} not found#{RESET}"
    end
  end

  def self.main
    ensure_dirs

    parser = OptionParser.new do |p|
      p.banner = "Usage: bytes [command] [options]"

      p.on("install NAME", "Install a library") { |name| install_lib(name) }
      p.on("remove NAME", "Remove a library") { |name| remove_lib(name) }

      p.on("plugin install NAME", "Install a plugin") { |name| install_plugin(name) }
      p.on("plugin remove NAME", "Remove a plugin") { |name| remove_plugin(name) }

      p.on("source install NAME", "Install a source") { |name| install_source(name) }
      p.on("source remove NAME", "Remove a source") { |name| remove_source(name) }
    end

    begin
      parser.parse
    rescue ex : OptionParser::InvalidOption
      puts "#{RED}Invalid command: #{ex.message}#{RESET}"
      puts parser
    rescue ex
      puts "#{RED}Error: #{ex.message}#{RESET}"
    end
  end
end

BytesCLI.main
