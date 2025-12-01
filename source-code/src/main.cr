require "http/client"
require "file_utils"
require "uri"
require "yaml"

module BytesManager
  @@home_dir = File.expand_path("~/.hackeros/hacker-lang")
  @@libs_dir = "#{@@home_dir}/libs"
  @@plugins_dir = "#{@@home_dir}/plugins"
  @@sources_dir = "#{@@home_dir}/sources"
  @@log_file = "/tmp/bytes-manager.log"
  LIBS_REPO_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
  PLUGINS_REPO_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/plugins-repo.hacker"
  GITHUB_REPO_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/github-repo.hacker"
  # stałe ścieżki w /tmp – repozytoria są zawsze trzymane tutaj
  LIBS_CACHE = "/tmp/bytes-libs.repo"
  PLUGINS_CACHE = "/tmp/bytes-plugins.repo"
  GITHUB_CACHE = "/tmp/bytes-github.repo"
  record LibInfo, url_template : String, versions : Array(String) do
    def latest : String
      versions.last
    end
  end
  def self.log(msg : String)
    File.open(@@log_file, "a") { |f| f.puts "#{Time.local} | #{msg}" }
  end
  def self.ensure_dirs
    FileUtils.mkdir_p(@@libs_dir)
    FileUtils.mkdir_p(@@plugins_dir)
    FileUtils.mkdir_p(@@sources_dir)
  end
  def self.download_repo(url : String, cache_file : String, force : Bool = false) : String
    if force && File.exists?(cache_file)
      File.delete(cache_file)
      log "Cache cleared: #{cache_file}"
    end
    unless File.exists?(cache_file)
      log "Downloading #{url} → #{cache_file}"
      resp = HTTP::Client.get(url)
      if resp.success?
        File.write(cache_file, resp.body)
      else
        puts "ERROR: Failed to download #{url} (#{resp.status_code})"
        exit 1
      end
    end
    File.read(cache_file)
  end
  def self.parse_repo(content : String) : Hash(String, LibInfo)
    yaml = YAML.parse(content)
    result = Hash(String, LibInfo).new
    %w[Public Community].each do |section|
      next unless yaml[section]?
      current_name = nil
      current_url = nil
      versions_list = [] of String
      yaml[section].as_h?.try &.each do |key_any, value_any|
        key = key_any.as_s
        if key == "versions"
          versions_list = value_any.as_a.map(&.to_s)
          versions_list.sort! { |a, b| version_compare(a, b) }
        else
          if current_name && current_url
            result[current_name] = LibInfo.new(current_url, versions_list.empty? ? [] of String : versions_list)
          end
          current_name = key
          current_url = value_any.as_s
          versions_list = [] of String
        end
      end
      if current_name && current_url
        result[current_name] = LibInfo.new(current_url, versions_list)
      end
    end
    result
  end
  def self.version_compare(a : String, b : String) : Int32
    va = a.split('.').map(&.to_i)
    vb = b.split('.').map(&.to_i)
    max = {va.size, vb.size}.max
    va += [0] * (max - va.size)
    vb += [0] * (max - vb.size)
    va.zip(vb).each do |aa, bb|
      return -1 if aa < bb
      return 1 if aa > bb
    end
    0
  end
  def self.get_libs(force = false) : Hash(String, LibInfo); parse_repo(download_repo(LIBS_REPO_URL, LIBS_CACHE, force)); end
  def self.get_plugins(force = false) : Hash(String, LibInfo); parse_repo(download_repo(PLUGINS_REPO_URL, PLUGINS_CACHE, force)); end
  def self.get_github_repos(force = false) : Hash(String, String)
    content = download_repo(GITHUB_REPO_URL, GITHUB_CACHE, force)
    yaml = YAML.parse(content)
    result = Hash(String, String).new
    %w[Public Community].each do |section|
      yaml[section]?.try &.as_h?.try &.each do |name_any, value|
        name = name_any.as_s
        url = value.as_s? || value.as_h?.try(&.["url"]?.try(&.as_s)) || ""
        result[name] = url unless url.empty?
      end
    end
    result
  end
  def self.refresh_libs; get_libs(true); puts "✓ Libraries repo refreshed (/tmp/bytes-libs.repo)"; end
  def self.refresh_plugins; get_plugins(true); puts "✓ Plugins repo refreshed (/tmp/bytes-plugins.repo)"; end
  def self.refresh_github
    get_github_repos(true)
    puts "✓ Github repo refreshed (/tmp/bytes-github.repo)"
    repos = get_github_repos
    return puts "Github repo is empty – nothing to build" if repos.empty?
    repos.each do |name, repo_url|
      dir = "#{@@sources_dir}/#{name}"
      git_url = repo_url.ends_with?(".git") ? repo_url : repo_url + ".git"
      if Dir.exists?(dir)
        log "git pull #{name}"
        `git -C "#{dir}" pull --quiet`
      else
        log "git clone #{git_url} #{dir}"
        `git clone "#{git_url}" "#{dir}"`
      end
      build_file = "#{dir}/build.hacker"
      if File.exists?(build_file)
        log "Building #{name} with hackerc build.hacker"
        `hackerc "#{build_file}"`
      end
    end
    puts "✓ Github repositories refreshed & built"
  end
  def self.install_library(name : String, ver : String? = nil)
    ensure_dirs
    libs = get_libs
    unless libs.has_key?(name)
      puts "Library '#{name}' not found in repository"
      return
    end
    info = libs[name]
    version = ver || info.latest
    unless info.versions.includes?(version)
      puts "Version #{version} not available for #{name} (available: #{info.versions.join(", ")})"
      return
    end
    url = info.url_template.gsub("{versions}", version)
    target_dir = "#{@@libs_dir}/#{name}/#{version}"
    if Dir.exists?(target_dir)
      puts "#{name} v#{version} already installed"
      return
    end
    FileUtils.mkdir_p(target_dir)
    asset_name = URI.parse(url).path.split("/").last
    temp_file = "/tmp/bytes-dl-#{asset_name}"
    log "Installing library #{name} v#{version}"
    HTTP::Client.get(url) do |resp|
      File.write(temp_file, resp.body_io)
    end
    if asset_name.ends_with?(".zip")
      `unzip -o "#{temp_file}" -d "#{target_dir}"`
    elsif asset_name.ends_with?(".tar.gz") || asset_name.ends_with?(".tgz")
      `tar -xzf "#{temp_file}" -C "#{target_dir}"`
    else
      FileUtils.cp(temp_file, "#{target_dir}/#{asset_name}")
    end
    File.delete(temp_file) if File.exists?(temp_file)
    puts "Installed #{name} v#{version}"
    log "Installed library #{name} v#{version}"
  end
  def self.install_plugin(name : String, ver : String? = nil)
    ensure_dirs
    plugins = get_plugins
    unless plugins.has_key?(name)
      puts "Plugin '#{name}' not found"
      return
    end
    info = plugins[name]
    version = ver || info.latest
    unless info.versions.includes?(version)
      puts "Version #{version} not available for plugin #{name}"
      return
    end
    target_dir = "#{@@plugins_dir}/#{name}"
    if Dir.exists?(target_dir)
      puts "Plugin #{name} already installed – use 'bytes update' or remove first"
      return
    end
    git_url = info.url_template
    git_url += ".git" unless git_url.ends_with?(".git")
    log "Installing plugin #{name} v#{version}"
    `git clone "#{git_url}" "#{target_dir}"`
    `git -C "#{target_dir}" checkout "v#{version}"` rescue `git -C "#{target_dir}" checkout "#{version}"`
    File.write("#{target_dir}/VERSION", version)
    puts "Installed plugin #{name} v#{version}"
    log "Installed plugin #{name} v#{version}"
  end
  def self.remove_library(name : String, ver : String? = nil)
    ensure_dirs
    base = "#{@@libs_dir}/#{name}"
    return puts "#{name} not installed" unless Dir.exists?(base)
    versions = Dir.children(base).sort { |a, b| version_compare(a, b) }
    version = ver || versions.last
    target = "#{base}/#{version}"
    return puts "Version #{version} not installed" unless Dir.exists?(target)
    FileUtils.rm_rf(target)
    Dir.delete(base) if Dir.empty?(base)
    puts "Removed #{name} v#{version}"
    log "Removed library #{name} v#{version}"
  end
  def self.remove_plugin(name : String)
    target = "#{@@plugins_dir}/#{name}"
    return puts "Plugin #{name} not installed" unless Dir.exists?(target)
    FileUtils.rm_rf(target)
    puts "Removed plugin #{name}"
    log "Removed plugin #{name}"
  end
  def self.update_all
    puts "Updating everything..."
    refresh_libs
    refresh_plugins
    refresh_github
    Dir.each_child(@@libs_dir) do |library_name|
      base = "#{@@libs_dir}/#{library_name}"
      next unless Dir.exists?(base)
      installed_versions = Dir.children(base).sort { |a, b| version_compare(a, b) }
      next if installed_versions.empty?
      latest_installed = installed_versions.last
      info = get_libs[library_name]?
      next unless info && version_compare(latest_installed, info.latest) < 0
      puts "Updating library #{library_name}: #{latest_installed} → #{info.latest}"
      install_library(library_name, info.latest)
    end
    Dir.each_child(@@plugins_dir) do |plugin_name|
      dir = "#{@@plugins_dir}/#{plugin_name}"
      next unless Dir.exists?(dir)
      current = File.read("#{dir}/VERSION") rescue nil
      next unless current
      info = get_plugins[plugin_name]?
      next unless info && version_compare(current, info.latest) < 0
      puts "Updating plugin #{plugin_name}: #{current} → #{info.latest}"
      remove_plugin(plugin_name)
      install_plugin(plugin_name, info.latest)
    end
    puts "Update completed!"
    log "Full update completed"
  end
  def self.clean_temp
    files = Dir["/tmp/bytes-*"] + [LIBS_CACHE, PLUGINS_CACHE, GITHUB_CACHE]
    files.each { |f| File.delete(f) if File.exists?(f) }
    puts "Cleaned temporary files and repo cache in /tmp/"
    log "Cleaned temp files and repo cache"
  end
  def self.usage
    puts "Bytes manager – menedżer bibliotek i pluginów Hacker Lang"
    puts "Usage: bytes <komenda> [opcje] [argumenty]"
    puts "Version: 0.1.0"
    puts "Commands:"
    puts "  install <library> [version]"
    puts "  remove <library> [version]"
    puts "  plugin install <plugin> [version]"
    puts "  plugin remove <plugin>"
    puts "  update"
    puts "  refresh [libs|plugins|github]"
    puts "  clean"
  end
  def self.run(argv : Array(String))
    if argv.empty?
      usage
      exit 0
    end
    cmd = argv.shift
    case cmd
    when "install"
      library = argv.shift? || begin
        puts "Missing library name"
        usage
        exit 1
      end
      version = argv.shift?
      install_library(library, version)
    when "remove"
      library = argv.shift? || begin
        puts "Missing library name"
        usage
        exit 1
      end
      version = argv.shift?
      remove_library(library, version)
    when "plugin"
      if argv.empty?
        puts "Missing subcommand for plugin"
        usage
        exit 1
      end
      sub = argv.shift
      case sub
      when "install"
        plugin = argv.shift? || begin
          puts "Missing plugin name"
          usage
          exit 1
        end
        version = argv.shift?
        install_plugin(plugin, version)
      when "remove"
        plugin = argv.shift? || begin
          puts "Missing plugin name"
          usage
          exit 1
        end
        remove_plugin(plugin)
      else
        puts "Unknown subcommand for plugin: #{sub}"
        usage
        exit 1
      end
    when "update"
      update_all
    when "refresh"
      if argv.empty?
        refresh_libs
        refresh_plugins
        refresh_github
        puts "✓ All repositories refreshed"
      else
        sub = argv.shift
        case sub
        when "libs"
          refresh_libs
        when "plugins"
          refresh_plugins
        when "github"
          refresh_github
        else
          puts "Unknown subcommand for refresh: #{sub}"
          usage
          exit 1
        end
      end
    when "clean"
      clean_temp
    else
      puts "Unknown command: #{cmd}"
      usage
      exit 1
    end
  end
end
BytesManager.run(ARGV)
