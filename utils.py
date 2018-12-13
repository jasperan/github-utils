#!/usr/bin/env python
import os
import time
import threading
import sys

class Committer(object):
	def __init__(self):
		pass

	def modifyFile(self, iteration):
		fd = open('a.txt', 'r+b')
		fd.write(str(iteration))
		fd.close()

	def commit(self, iteration):
		os.system('git add .')
		os.system("git commit -m 'Commit corresponding to iteration %s'" % str(iteration))
		if iteration % 50000 == 0 and iteration != 0:
			self.push()

	def push(self):
		os.system('git push')

	def push_upstream(self, iteration):
		os.system('git push --set-upstream origin %s' % str(iteration))

	def branch(self, iteration):
		os.system('git branch %s' % str(iteration))
		os.system('git add .')
		os.system("git commit -m 'Commit corresponding to iteration %s'" % str(iteration))
		self.push_upstream(iteration)
	
	def checkout(self, iteration):
		self.modifyFile(iteration)
		os.system('git branch %s' % str(iteration))
		os.system('git checkout %s' % str(iteration))
		os.system('git add .')
		os.system("git commit -m 'Commit corresponding to iteration %s'" % str(iteration))
		self.push_upstream(iteration)

	def tag(self, iteration):
		os.system("git tag -a 1.0.%s -m 'Release of version 1.0.%s" % (iteration, iteration))
		os.system('git push --tags')

def branch(i):
	obj = Committer()
	while True:
		obj.modifyFile(i)
		obj.checkout(i)
		i = i + 1

def commit(i):
	obj = Committer()
	while True:
		obj.modifyFile(i)
		obj.commit(i)
		i = i - 1
		if i == 0:
			sys.exit(1)

def tcommit(numthreads):
	threads = list()
	stringlist = list()
	for i in range(numthreads):
		stringlist.append('thread%s' % i)
	# Now we have a list of files and a set of fil
	for i in range(len(stringlist)):
		t = threading.Thread(target=threadtcommit, args=[i])
		threads.append(t)
		t.start()

def threadtcommit(number):
	obj = Committer()
	i = 0
	while True:
		fd = open('thread%s.txt' % number, 'r+b') # thread0.txt... threadn.txt
		fd.write('%s' % number)
		fd.close()
		obj.commit(i)
		i = i + 1

def tag(number):
	obj = Committer()
	while True:
		obj.tag(i)
		i = i + 1

def release(number):
	pass

def main():
	script, action, goal = sys.argv
	if goal == 'inf':
		goal = 10000000
	if action == 'branch':
		branch(int(goal))
	elif action == 'commit':
		commit(int(goal))
	elif action =='tcommit':
		tcommit(int(8))
	elif action == 'tag':
		tag(int(goal))
	else:
		print 'Invalid input. Choose between branch and commit, with an iteration number.'
		sys.exit(1)

	"""
	threads = list()
	t = threading.Thread(target=worker)
	threads.append(t)
	t.start()
	"""

if __name__ == '__main__':
	main()