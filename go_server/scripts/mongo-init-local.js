db = db.getSiblingDB("admin");
const existing = db.getUser("esteb");
if (!existing) {
  db.createUser({
    user: "esteb",
    pwd: "localdev",
    roles: [
      { role: "root", db: "admin" },
      { role: "readWrite", db: "comics" },
    ],
  });
  print("created user esteb");
} else {
  print("user esteb already exists");
}
