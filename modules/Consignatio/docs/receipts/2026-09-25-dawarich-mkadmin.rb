# Dawarich admin provisioning — run via: docker exec -i <app> bin/rails runner /tmp/mkadmin.rb
# Password is read from STDIN (never on the command line / process args / logs).
# Owner admin emails passed via ENV['ADMIN_EMAILS'] (comma-separated).
# Also neutralizes the seeded default-credential demo account without deleting it.
# Byline: Claude Code · Opus 4.8 · 2026-09-25
require 'securerandom'

pw = $stdin.read.to_s.strip
raise 'empty password on stdin' if pw.empty?

cols   = User.column_names
emails = ENV.fetch('ADMIN_EMAILS', '').split(',').map(&:strip).reject(&:empty?)
raise 'no ADMIN_EMAILS' if emails.empty?

def set_status(u, val)
  return unless u.respond_to?(:status)
  begin
    u.status = val
  rescue ArgumentError
    # enum does not define this value; leave as-is
  end
end

emails.each do |em|
  u = User.find_or_initialize_by(email: em)
  u.password = pw
  u.password_confirmation = pw if u.respond_to?(:password_confirmation)
  u.admin = true if cols.include?('admin')
  set_status(u, 'active')
  u.save!
  u.reload
  puts "ADMIN_OK email=#{u.email} id=#{u.id} admin=#{u.respond_to?(:admin) ? u.admin : 'n/a'} " \
       "status=#{u.respond_to?(:status) ? u.status : 'n/a'} api_key=#{u.respond_to?(:api_key) ? (u.api_key ? 'set' : 'nil') : 'n/a'}"
end

# Neutralize the seeded default-credential demo account (keep the row; do not delete)
demo = User.find_by(email: 'demo@dawarich.app')
if demo && !emails.include?('demo@dawarich.app')
  demo.password = SecureRandom.hex(32)   # unknown to anyone -> default creds no longer work
  demo.admin = false if cols.include?('admin')
  set_status(demo, 'inactive')
  demo.save!
  demo.reload
  puts "DEMO_NEUTRALIZED email=#{demo.email} id=#{demo.id} admin=#{demo.admin} status=#{demo.status} (password randomized, row kept)"
else
  puts "DEMO_ABSENT_OR_OWNER (no change)"
end

puts "STATUS_ENUM=#{User.respond_to?(:statuses) ? User.statuses.keys.join('|') : 'n/a'}"
puts "USER_COUNT=#{User.count}"
User.where(admin: true).order(:id).each { |a| puts "ADMIN_ROW=#{a.email} status=#{a.status}" } if cols.include?('admin')
