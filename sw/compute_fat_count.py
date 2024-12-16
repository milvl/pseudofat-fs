import csv

MAX_FS_SIZE = 4294967295  # 4 GB
CLUSTER_SIZE = 4000  # 4 KB
FS_STRUCT_SIZE = 32  # Size of FileSystem struct
FAT_ROW_SIZE = 4

with open('fat_count.csv', mode='w') as file:
    writer = csv.writer(file)
    writer.writerow(['i', 'Cluster Space', 'Both FAT Size', 'Cluster Count'])

    # iterative calculation for FAT size and clusters
    available_space = MAX_FS_SIZE - FS_STRUCT_SIZE
    fat_table_size = 0
    i = 0

    while True:
        cluster_count = (available_space - fat_table_size ) // CLUSTER_SIZE

        fat_table_size = cluster_count * FAT_ROW_SIZE

        new_available_space = MAX_FS_SIZE - FS_STRUCT_SIZE - (fat_table_size * 2)

        if new_available_space == available_space:
            writer.writerow([i, cluster_count * CLUSTER_SIZE, fat_table_size * 2, cluster_count])
            break

        writer.writerow([i, cluster_count * CLUSTER_SIZE, fat_table_size * 2, cluster_count])

        available_space = new_available_space
        i += 1
