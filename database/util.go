package database

func removeDuplicates(slice []string) []string {
  allKeys := make(map[string]bool)
  var unique []string
  for _, item := range slice {
    if _, value := allKeys[item]; !value {
      allKeys[item] = true
      unique = append(unique, item)
    }
  }
  return unique
}
