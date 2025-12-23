import asyncio
import aiohttp

URL = "http://localhost/auth/login"
USERS = 10
REQUESTS_PER_USER = 100

async def abuse_user(user_id):
    async with aiohttp.ClientSession() as session:
        for i in range(REQUESTS_PER_USER):
            payload = {
                "username": "test1",
                "password": "test123"
            }

            async with session.post(URL, json=payload) as resp:
                print(f"user-{user_id} -> {resp.status}")


async def main():
    tasks = []
    for i in range(USERS):
        tasks.append(abuse_user(i))

    await asyncio.gather(*tasks)

asyncio.run(main())
