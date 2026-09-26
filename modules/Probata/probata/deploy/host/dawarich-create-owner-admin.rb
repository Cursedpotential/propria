# Byline: Claude Code · Opus 5.5 · 2026-09-25
# Create/update the owner's admin login on the ovh-files Dawarich service (Coolify service t2awforcbp4hlskjnvwqovzb).
# Owner rule: owner admin on every service, standard password (~/.secrets/owner-login.env OWNER_STD_PASSWORD).
# Dawarich logs in by EMAIL, so the owner login is ADMIN_EMAIL (msalem85@salem.local).
# Also rotates the seeded demo@dawarich.app account's public default password so it cannot be used.
# Run (password on stdin, never in a command line):
#   ssh root@ovh-files "docker exec -i <dawarich_app> sh -c 'cat > /var/app/create_owner_admin.rb'" < this-file
#   printf '%s\n' "$PW" | ssh root@ovh-files 'read -r PW; docker exec -e ADMIN_EMAIL=msalem85@salem.local -e ADMIN_PW="$PW" <dawarich_app> bin/rails runner /var/app/create_owner_admin.rb'
email = ENV.fetch("ADMIN_EMAIL"); pw = ENV.fetch("ADMIN_PW")
u = User.find_or_initialize_by(email: email)
u.password = pw
u.password_confirmation = pw if u.respond_to?(:password_confirmation=)
u.admin = true
u.first_name = "Matt" if u.respond_to?(:first_name=) && u.first_name.to_s.empty?
u.last_name  = "Salem" if u.respond_to?(:last_name=) && u.last_name.to_s.empty?
if u.respond_to?(:status=) && User.respond_to?(:statuses) && User.statuses.key?("active")
  u.status = "active"
end
unless u.save
  puts "SAVE FAILED: #{u.errors.full_messages.join('; ')}"
  exit 1
end
puts({email: u.email, admin: u.admin, status: (u.status rescue nil), password_ok: u.valid_password?(pw)}.inspect)
d = User.find_by(email: "demo@dawarich.app")
if d
  d.password = SecureRandom.hex(24)
  d.password_confirmation = d.password if d.respond_to?(:password_confirmation=)
  d.save!(validate: false)
  puts "demo account: default password rotated"
end
