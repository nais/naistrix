Type "inventory list"
Enter
Wait+Screen /Gateway docs/
Wait
Sleep 1.5s

Type "inventory show unknown --error-format json"
Enter
Wait+Screen /service_not_found/
Wait
Sleep 2s
