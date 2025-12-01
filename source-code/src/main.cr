require "clim"
require "http/client"
require "file_utils"
require "uri"
require "yaml"

class Bytes < Clim
  @home_dir = File.expand_path("~/.hackeros/hacker-lang")
  @libs_dir = "#{@home_dir}/libs"
  @plugins_dir = "#{@home_dir}/plugins"
  @sources_dir = "#{@home_dir}/sources"
  @cache_dir = "#{@home_dir}/bytes-cache"
  @log_file = "/tmp/bytes-manager.log"

  LIBS_REPO_URL    = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/bytes.io"
  PLUGINS_REPO_URL = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/plugins-repo.hacker"
  GITHUB_REPO_URL  = "https://raw.githubusercontent.com/Bytes-Repository/bytes.io/main/repository/github-repo.hacker"

  record LibInfo, url_template : String, versions : Array(String) do
    def latest = versions.sort.last
  end

  def log(msg : String)
    File.open(@log_file, "a") { |f| f.puts "#{Time.local} | #{msg}" }
  end

  def ensure_dirs
    FileUtils.mkdir_p(@libs_dir)
    FileUtils.mkdir_p(@plugins_dir)
    FileUtils.mkdir_p(@sources_dir)
    FileUtils.mkdir_p(@cache_dir)
  end

  def download_repo(url : String, force : Bool = false) : String
    filename = URI.parse(url).path.split("/").last
    cache_file = "#{@cache_dir}/#{filename}"

    if force || !File.exists?(cache_file)
      log "Downloading #{url}"
      resp = HTTP::Client.get(url)
      if resp.success?
        File.write(cache_file, resp.body)
      else
        puts "ERROR: Failed to download #{url}"
        exit 1
      end
    end
    File.read(cache_file)
  end

  def parse_repo(content : String) : Hash(String, LibInfo)
    yaml = YAML.parse(content)
    result = Hash(String, LibInfo).new

    %w[Public Community].each do |section|
      next unless yaml[section]?
      yaml[section].as_h?.try &.each do |name_any, info_any|
        name = name_any.as_s
        info = info_any.as_h?
        next unless info && info["url"]? && info["versions"]?

        url_template = info["url"].as_s
        versions = info["versions"].as_a?.try(&.map(&.as_s)) || [] of String
        versions.sort!
        result[name] = LibInfo.new(url_template, versions)
      end
    end
    result
  end

  def get_libs(force = false)    : Hash(String, LibInfo); parse_repo(download_repo(LIBS_REPO_URL, force)); end
  def get_plugins(force = false) : Hash(String, LibInfo); parse_repo(download_repo(PLUGINS_REPO_URL, force)); end

  def get_github_repos(force = false) : Hash(String, String)
    content = download_repo(GITHUB_REPO_URL, force)
    yaml = YAML.parse(content)
    result = Hash(String, String).new
    %w[Public Community].each do |section|
      yaml[section]?.try &.as_h?.try &.each do |name_any, value|
        name = name_any.as_s
        url = value.as_s? || value.as_h?.try(&["url"]?.try(&.as_s))
        result[name] = url if url && !url.empty?
      end
    end
    result
  end

  def refresh_libs;    get_libs(true);    puts "✓ Libraries repo refreshed";    end
  def refresh_plugins; get_plugins(true); puts "✓ Plugins repo refreshed";     end

  def refresh_github
    repos = get_github_repos(true)
    return puts "✓ Github repo is empty – nothing to do" if repos.empty?

    repos.each do |name, repo_url|
      dir = "#{@sources_dir}/#{name}"
      git_url = repo_url.ends_with?(".git") ? repo_url : repo_url + ".git"

      if Dir.exists?(dir)
        log "git pull #{name}"
        Process.run("git", ["-C", dir, "pull", "--quiet"])
      else
        log "git clone #{name}"
        Process.run("git", ["clone", git_url, dir])
      end

      build_file = "#{dir}/build.hacker"
      if File.exists?(build_file)
        log "Building #{name} with hackerc"
        Process.run("hackerc", [build_file], chdir: dir)
      end
    end
    puts "✓ Github repositories refreshed & built"
  end

  def install_library(name : String, ver : String? = nil)
    ensure_dirs
    libs = get_libs
    unless libs.has_key?(name)
      puts "Library '#{name}' not found in repository"
      return
    end

    info = libs[name]
    version = ver || info.latest
    unless info.versions.includes?(version)
      puts "Version #{version} not available for #{name}"
      return
    end

    url = info.url_template.gsub("{versions}", version)
    target_dir = "#{@libs_dir}/#{name}/#{version}"

    if Dir.exists?(target_dir)
      puts "#{name} v#{version} already installed"
      return
    end

    FileUtils.mkdir_p(target_dir)
    asset_name = URI.parse(url).path.split("/").last
    temp_file = "/tmp/bytes-#{asset_name}"

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

  def install_plugin(name : String, ver : String? = nil)
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

    target_dir = "#{@plugins_dir}/#{name}"
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

  def remove_library(name : String, ver : String? = nil)
    ensure_dirs
    base = "#{@libs_dir}/#{name}"
    return puts "#{name} not installed" unless Dir.exists?(base)

    versions = Dir.children(base).sort
    version = ver || versions.last

    target = "#{base}/#{version}"
    return puts "Version #{version} not installed" unless Dir.exists?(target)

    FileUtils.rm_rf(target)
    Dir.delete(base) if Dir.empty?(base)

    puts "Removed #{name} v#{version}"
    log "Removed library #{name} v#{version}"
  end

  def remove_plugin(name : String, _ver : String? = nil)
    target = "#{@plugins_dir}/#{name}"
    return puts "Plugin #{name} not installed" unless Dir.exists?(target)

    FileUtils.rm_rf(target)
    puts "Removed plugin #{name}"
    log "Removed plugin #{name}"
  end

  def update_all
    puts "Updating everything..."
    refresh_libs
    refresh_plugins
    refresh_github

    # update libs
    Dir.each_child(@libs_dir) do |lib|
      base = "#{@libs_dir}/#{lib}"
      next unless Dir.exists?(base)
      installed = Dir.children(base).sort
      next if installed.empty?

      latest_installed = installed.last
      info = get_libs[lib]?
      next unless info && info.latest != latest_installed

      puts "Updating library #{lib}: #{latest_installed} → #{info.latest}"
      install_library(lib, info.latest)
    end

    # update plugins
    Dir.each_child(@plugins_dir) do |plugin|
      dir = "#{@plugins_dir}/#{plugin}"
      next unless Dir.exists?(dir)
      current = File.read("#{dir}/VERSION") rescue nil
      next unless current

      info = get_plugins[plugin]?
      next unless info && info.latest != current

      puts "Updating plugin #{plugin}: #{current} → #{info.latest}"
      remove_plugin(plugin)
      install_plugin(plugin, info.latest)
    end

    puts "Update completed!"
    log "Full update completed"
  end

  def clean_temp
    Dir["/tmp/bytes-*"].each { |f| File.delete(f) if File.file?(f) }
    puts "Cleaned temporary files"
    log "Cleaned temp files"
  end

  # CLI definition
  main do
    desc "Bytes manager – narzędzie do zarządzania bibliotekami i pluginami Hacker Lang"
    usage "bytes <komenda> [opcje] [argumenty]"
    version "0.1.0"

    command "install" do
      usage "install <biblioteka> [wersja]"
      arg "library", required: true
      arg "version", required: false
      run do |opts, args|
        install_library(args.library, args.version?)
      end
    end

    command "remove" do
      usage "remove <biblioteka> [wersja]   (bez wersji → usuwa najnowszą)"
      arg "library", required: true
      arg "version", required: false
      run do |opts, args|
        remove_library(args.library, args.version?)
      end
    end

    command "plugin" do
      command "install" do
        usage "plugin install <plugin> [wersja]"
        arg "plugin", required: true
        arg "version", required: false
        run do |opts, args|
          install_plugin(args.plugin, args.version?)
        end
      end

      command "remove" do
        usage "plugin remove <plugin>"
        arg "plugin", required: true
        run do |opts, args|
          remove_plugin(args.plugin)
        end
      end
    end

    command "update" do
      run do
        update_all
      end
    end

    command "refresh" do
      run do
        # bez argumentu → wszystko
        refresh_libs
        refresh_plugins
        refresh_github
      end

      command "libs" do
        run { refresh_libs }
      end

      command "plugins" do
        run { refresh_plugins }
      end

      command "github" do
        run { refresh_github }
      end
    end

    command "clean" do
      run { clean_temp }
    end
  end
end

Bytes.start(ARGV)
