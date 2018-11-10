local f = load("print('hello')")
f()
--> =hello

do
    local env = {x = 1}
    local f = load("x = 2", "chunk", "bt", env)
    print(env.x)
--> =1
    f()
    print(env.x)
--> =2
end

load("print(...)")(1, 2)
--> =1	2

-- This loads and executes the given file
loadfile("lua/loadfile.lua.notest")()
--> =loadfile

print(pcall(loadfile, "lua/nonexistent_file"))
--> ~^false	.*

dofile("lua/loadfile.lua.notest")
--> =loadfile

load(coroutine.wrap(function ()
    coroutine.yield("print(")
    coroutine.yield("'hello')")
end))()
--> =hello

load(coroutine.wrap(function ()
    coroutine.yield("print(")
    coroutine.yield("'abc')")
    coroutine.yield("")
    coroutine.yield("print(")
    coroutine.yield("'xyz')")
end))()
--> =abc

print(load(function() error("argh") end))
--> ~nil\t.*argh

print(load(function() return {} end))
--> ~nil\t.*must return a string

print(pcall(load, {}))
--> ~false\t.*string or function

print(load("\x00", "", "t"))
--> ~nil\t.*binary chunk

local z = "haha"
load(string.dump(function() print("hi", z) end))()
--> =hi	nil
